module.exports = function (grunt) {
    grunt.initConfig({
        jshint: {
            files: ['Gruntfile.js', 'js/scripts.js'],
            options: {
                globals: {
                    jQuery: true
                }
            }
        },
        watch: {
            scripts: {
                files: ['<%= jshint.files %>'],
                tasks: ['jshint', 'uglify', 'clean:js', 'rev']
            },
            stylesheets: {
                files: ['css/style.scss', 'css/solarizeddark.scss', 'css/nanoscroller.css'],
                tasks: ['sass', 'cssmin', 'clean:css', 'rev']
            }
        },
        uglify: {
            target: {
                files: {
                    'js/www.js': ['js/scripts.js'],
                    'js/where.js': ['js/topojson.v1.min.js']
                }
            },
            options: {
                mangle: false,
                compress: false,
                beautify: false,
                sourceMap: true
            }
        },
        rev: {
            options: {
                algorithm: 'md5',
                length: 16
            },
            assets: {
                files: [{
                    src: ['js/www.js', 'js/where.js', 'css/www.css']
                }]
            }
        },
        compress: {
            assets: {
                options: {
                    archive: 'assets.tgz'
                },
                files: [
                    {
                        src: ['fonts/*', 'js/*.www.js', 'js/*.where.js', 'js/world-50m.json', 'css/*.www.css'],
                        dest: '.'
                    }
                ]
            }
        },
        sass: {
            target: {
                files: {
                    'css/style.css': 'css/style.scss'
                }
            }
        },
        cssmin: {
            combine: {
                files: {
                    'css/www.css': ['css/style.css', 'css/solarizeddark.css', 'css/nanoscroller.css']
                }
            }
        },
        clean: {
            css: [
                'css/*.www.css'
            ],
            js: [
                'js/*.www.js',
                'js/*.where.js'
            ]
        }
    });

    grunt.loadNpmTasks('grunt-contrib-jshint');
    grunt.loadNpmTasks('grunt-contrib-watch');
    grunt.loadNpmTasks('grunt-contrib-uglify');
    grunt.loadNpmTasks('grunt-rev');
    grunt.loadNpmTasks('grunt-sass');
    grunt.loadNpmTasks('grunt-contrib-cssmin');
    grunt.loadNpmTasks('grunt-contrib-clean');
    grunt.loadNpmTasks('grunt-contrib-compress');

    grunt.registerTask('default', ['jshint', 'uglify', 'sass', 'cssmin', 'clean', 'rev', 'compress']);

};
