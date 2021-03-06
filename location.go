package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/kpawlik/geojson"
	"github.com/martinlindhe/unit"
	"strconv"
	"time"
)

/*
This should be some sort of thing that's sent from the phone
*/
type Location struct {
	Latitude             float64 `json:"lat" binding:"required"`
	Longitude            float64 `json:"long" binding:"required"`
	Timestamp            time.Time
	DeviceTimestamp      time.Time
	DeviceTimestampAsInt int64   `json:"time" binding:"required"`
	Accuracy             float32 `json:"acc" binding:"required"`
	Distance             float64
	GSMType              string `json:"gsmtype" binding:"required"`
	WifiSSID             string `json:"wifissid" binding:"required"`
	DeviceID             string `json:"deviceid" binding:"required"`
	Geocoding            string
}

func GetLastLoction() (*Location, error) {
	var location Location
	defer timeTrack(time.Now())
	err := db.QueryRow("select geocoding, ST_Y(ST_AsText(point)),ST_X(ST_AsText(point)),devicetimestamp from locations where geocoding is not null order by devicetimestamp desc limit 1").Scan(&location.Geocoding, &location.Latitude, &location.Longitude, &location.Timestamp)
	return &location, err
}

/*
In miles.
*/
func GetTotalDistance(year int) (float64, error) {
	var distance float64
	defer timeTrack(time.Now())
	err := db.QueryRow("select sum(distance) from (select ST_Distance(point,lag(point) over (order by devicetimestamp asc)) as distance from locations where date_part('year'::text, date(devicetimestamp at time zone 'UTC')) = $1) a;", time.Now().UTC().Year()).Scan(&distance)
	if err != nil {
		return 0, err
	}
	distanceInMeters := unit.Length(distance) * unit.Meter
	return distanceInMeters.Miles(), nil
}

func GetLineStringAsJSON(year int) (string, error) {
	lineString := geojson.NewLineString(nil)
	sqlStatement := "select ST_X(ST_AsText(point)),ST_Y(ST_AsText(point)),ST_Distance(point,lag(point) over (order by devicetimestamp asc)) from locations where date_part('year'::text, date(devicetimestamp at time zone 'UTC'))=$1 and accuracy<(select percentile_disc(0.9) within group (order by accuracy) from locations where date_part('year'::text, date(devicetimestamp at time zone 'UTC'))=$1) order by devicetimestamp asc"
	rows, err := db.Query(sqlStatement, year)
	if err != nil {
		return "", err
	}

	defer rows.Close()
	for rows.Next() {
		var coords geojson.Coordinate
		coords = geojson.Coordinate{0, 0}
		var distance float32
		rows.Scan(&coords[0], &coords[1], &distance)
		if distance > 100 {
			// We only want to add points where something's actually moved significantly, this is in metres
			lineString.AddCoordinates(coords)
		}
	}
	// Dump the stuff into some sort of geojson thingie
	feature := geojson.NewFeature(lineString, nil, nil)
	featureCollection := geojson.NewFeatureCollection([]*geojson.Feature{feature})
	json, err := geojson.Marshal(featureCollection)
	if err != nil {
		return "", err
	}
	return json, nil
}

/* HTTP handlers */
func WhereLineStringHandler(c *gin.Context) {
	yearstring := c.Params.ByName("year")
	year, err := strconv.Atoi(yearstring)
	if err != nil {
		c.String(400, fmt.Sprintf("%v does not look like a year", yearstring))
	}
	linestring, err := GetLineStringAsJSON(year)
	if err != nil {
		InternalError(err)
		c.String(500, "Internal Error\n"+err.Error())
	}
	c.Data(200, "application/json", []byte(linestring))
}

/*
Receive POST from phone. This should be an application/json containing an array of points.
*/

func LocatorHandler(c *gin.Context) {
	c.String(204, "Deprecated")
	return
}

func LocationHandler(c *gin.Context) {
	thisyear := time.Now().UTC().Year()
	location, err := GetLastLoction()
	distance, err := GetTotalDistance(thisyear)
	if err != nil {
		c.String(500, err.Error())
	}
	c.Header("Last-modified", location.Timestamp.Format("Mon, 02 Jal 2006 15:04:05 GMT"))
	c.JSON(200, gin.H{
		"name":          location.Name(),
		"latitude":      fmt.Sprintf("%.2f", location.Latitude),
		"longitude":     fmt.Sprintf("%.2f", location.Longitude),
		"totalDistance": humanize.FormatFloat("#,###.##", distance),
	})
}

func LocationHeadHandler(c *gin.Context) {
	location, err := GetLastLoction()
	if err != nil {
		c.String(500, err.Error())
	}
	c.Header("Last-modified", location.Timestamp.Format("Mon, 02 Jal 2006 15:04:05 GMT"))
	c.Status(200)
}
