package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/growse/concurrent-expiring-map"
	"github.com/lib/pq"
	"github.com/mailgun/mailgun-go"
	"github.com/oxtoacart/bpool"
	"gopkg.in/fsnotify.v1"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"
)

var (
	db                 *sql.DB
	stylesheetfilename string
	javascriptfilename string
	configuration      Configuration
	gun                mailgun.Mailgun
	templates          *template.Template
	bufPool            *bpool.BufferPool
	memoryCache        cmap.ConcurrentMap
)

type Configuration struct {
	DbUser             string
	DbName             string
	DbPassword         string
	DbHost             string
	TemplatePath       string
	StaticPath         string
	CpuProfile         string
	GeocodeApiURL      string
	MailgunKey         string
	Production         bool
	DefaultCacheExpiry time.Duration
}

type ArticleMonth struct {
	FirstOfTheYear bool
	Year           int
	Month          time.Month
	Count          int
}

var funcMap = template.FuncMap{
	"RenderFloat": RenderFloat,
}

func RobotsHandler(c *gin.Context) {
	c.File(path.Join(configuration.TemplatePath, "robots.txt"))
}

func LoadTemplates() {
	templateGlob := path.Join(configuration.TemplatePath, "*.html")
	templates = template.Must(template.New("Yay Templates").Funcs(funcMap).ParseGlob(templateGlob))
}

func InternalError(err error) {
	log.Printf("%v", err)
	debug.PrintStack()
	if configuration.Production {
		m := mailgun.NewMessage("Sender <blogbot@growse.com>", "ERROR: www.growse.com", fmt.Sprintf("%v\n%v", err, string(debug.Stack())), "sysadmin@growse.com")
		response, id, _ := gun.Send(m)
		fmt.Printf("Response ID: %s\n", id)
		fmt.Printf("Message from server: %s\n", response)
	}
}

func main() {
	//Flags
	configFile := flag.String("configFile", "config.json", "File path to the JSON configuration")
	flag.Parse()

	//Config parsing
	file, err := os.Open(*configFile)
	if err != nil {
		log.Fatalf("Unable to open configuration file: %v", err)
	}

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&configuration)
	if err != nil {
		log.Fatalf("Unable to parse configuration file: %v", err)
	}

	if configuration.TemplatePath == "" {
		log.Fatalf("No template directory supplied")
	}
	if configuration.StaticPath == "" {
		log.Fatalf("No static directory supplied")
	}
	if _, err := os.Stat(configuration.TemplatePath); os.IsNotExist(err) {
		log.Fatalf("No such file or directory: %s", configuration.TemplatePath)

	}
	if _, err := os.Stat(configuration.StaticPath); os.IsNotExist(err) {
		log.Fatalf("No such file or directory: %s", configuration.StaticPath)
	}

	gun = mailgun.NewMailgun("valid-mailgun-domain", "private-mailgun-key", "public-mailgun-key")

	//Initialize the template output buffer pool
	bufPool = bpool.NewBufferPool(16)

	//Get around auto removing of pq
	yay := pq.ListenerEventConnected
	log.Printf("Here's the number zero: %v", yay)

	//Database time

	connectionString := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", configuration.DbHost, configuration.DbUser, configuration.DbName, configuration.DbPassword)
	db, err = sql.Open("postgres", connectionString)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	//Get the router
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	//Load the templates. Don't use gin for this, because we want to render to a buffer later
	LoadTemplates()
	//Static is over here
	router.Static("/static/", configuration.StaticPath)

	//Get latest updated stylesheet
	stylesheets, _ := ioutil.ReadDir(path.Join(configuration.StaticPath, "css"))
	var lastTimeCss time.Time
	for _, file := range stylesheets {
		if !file.IsDir() && (lastTimeCss.IsZero() || file.ModTime().After(lastTimeCss)) && strings.HasSuffix(file.Name(), ".www.css") {
			lastTimeCss = file.ModTime()
			stylesheetfilename = file.Name()
		}
	}
	if stylesheetfilename == "" {
		log.Fatal("No stylesheet found in staticpath. Perhaps run Grunt first?")
	}
	//Get the latest javascript file
	javascripts, _ := ioutil.ReadDir(path.Join(configuration.StaticPath, "js"))
	var lastTimeJs time.Time

	for _, file := range javascripts {
		if !file.IsDir() && (lastTimeJs.IsZero() || file.ModTime().After(lastTimeJs)) && strings.HasSuffix(file.Name(), ".www.js") {
			lastTimeJs = file.ModTime()
			javascriptfilename = file.Name()
		}
	}
	if javascriptfilename == "" {
		log.Fatal("No javascript found in staticpath. Perhaps run Grunt first?")
	}

	//Detect changes to css / js and update those paths.
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		defer watcher.Close()
		watcher.Add(path.Join(configuration.StaticPath, "css"))
		watcher.Add(path.Join(configuration.StaticPath, "js"))
		watcher.Add(configuration.TemplatePath)
		go func() {
			for {
				select {
				case event := <-watcher.Events:
					if event.Op&fsnotify.Create == fsnotify.Create {
						if strings.HasSuffix(event.Name, ".www.css") {
							log.Printf("New CSS Detected: %s", path.Base(event.Name))
							stylesheetfilename = path.Base(event.Name)
							memoryCache.Flush()

						}
						if strings.HasSuffix(event.Name, ".www.js") {
							log.Printf("New JS Detected: %s", path.Base(event.Name))
							javascriptfilename = path.Base(event.Name)
							memoryCache.Flush()
						}
					} else if strings.HasSuffix(event.Name, ".html") {
						log.Printf("Template changed: %s", path.Base(event.Name))
						LoadTemplates()
						memoryCache.Flush()
					}
				case err := <-watcher.Errors:
					log.Println("error:", err)
				}
			}
		}()
	} else {
		log.Printf("Initify watcher failed: %s Continuing.", err)
	}

	//Caching time
	memoryCache = cmap.New()

	//Cpu Profiling time
	if configuration.CpuProfile != "" {
		f, err := os.Create(configuration.CpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	//Catch SIGTERM to stop the profiling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("captured %v, stopping profiler and exiting..", sig)
			pprof.StopCPUProfile()
			os.Exit(1)
		}
	}()

	//Ugly hack to deal with the fact that httprouter can't cope with both /static/ and /:year existing
	//All years will begin with 2. So this sort of helps. Kinda.
	router.GET("/2:year/:month/", MonthHandler)
	router.GET("/2:year/:month/:day/:slug/", ArticleHandler)
	router.GET("/rss/", RSSHandler)
	router.GET("/where/", WhereHandler)
	router.GET("/where/linestring/:year/", WhereLineStringHandler)
	router.GET("/", LatestArticleHandler)
	router.GET("/robots.txt", RobotsHandler)
	router.GET("/news/rss/", func(c *gin.Context) { c.Redirect(301, "/rss/") })
	router.GET("/sitemap.xml", UncompressedSiteMapHandler)
	router.GET("/sitemap.xml.gz", CompressedSiteMapHandler)
	router.POST("/search/", SearchPostHandler)
	router.POST("/locator/", LocatorHandler)
	router.GET("/search/:searchterm/", SearchHandler)
	router.Run(":8080")
}
