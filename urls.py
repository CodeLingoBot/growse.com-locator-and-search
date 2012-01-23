from django.conf.urls.defaults import *
from django.views.generic.simple import direct_to_template
from blog.rssfeed import RssFeed
from django.contrib import admin
admin.autodiscover()

urlpatterns = patterns('',
    (r'^robots\.txt$',direct_to_template,{'template':'robots.txt','mimetype':'text/plain'}),
    (r'^cp/', include(admin.site.urls)),
    (r'^news/rss/$',RssFeed()),
    (r'^news/comments/(?P<article_shorttitle>.+)/$','blog.views.article'),
    (r'^news/archive/$','blog.views.newsarchive'),
    (r'^news/archive/(?P<newsarchive_year>\d{4})/(?P<newsarchive_month>\d{2})/$','blog.views.newsarchive_month'),
    (r'^projects/(?P<article_shorttitle>.+)/$','blog.views.article'),
    (r'^misc/(?P<article_shorttitle>.+)/$','blog.views.article'),
    (r'videos/(?P<article_shorttitle>.+)/$','blog.views.article'),
    (r'^news/$', 'blog.views.newsindex'),
    (r'^photos/$', 'blog.views.photos'),
    (r'^projects/$', 'blog.views.projectsindex'),
    (r'^videos/$', 'blog.views.videosindex'),
    (r'^misc/$', 'blog.views.miscindex'),
    (r'^links/$', 'blog.views.links'),
    (r'^search/(?P<searchterm>.+)/(?P<page>\d+)/$','blog.views.search'),
    (r'^search/(?P<searchterm>.+)/$','blog.views.search'),
    (r'^search/$','blog.views.search'),
    (r'^$','blog.views.frontpage'),
)
