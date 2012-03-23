from django.conf.urls import *
from django.views.generic.simple import direct_to_template
from growse_com.blog.rssfeed import RssFeed
from django.contrib import admin
from sitemaps import BlogSitemap,FlatSitemap
admin.autodiscover()
sitemaps = {
	'blog': BlogSitemap,
	'flat': FlatSitemap,
	}

urlpatterns = patterns('',
    (r'^robots\.txt$',direct_to_template,{'template':'robots.txt','mimetype':'text/plain'}),
    (r'^sitemap\.xml$', 'django.contrib.sitemaps.views.sitemap', {'sitemaps': sitemaps}),
    (r'^cp/', include(admin.site.urls)),
    (r'^news/rss/$',RssFeed()),
    (r'^news/comments/(?P<article_shorttitle>.+)/$','growse_com.blog.views.article'),
    (r'^news/archive/$','growse_com.blog.views.newsarchive'),
    (r'^news/archive/(?P<newsarchive_year>\d{4})/(?P<newsarchive_month>\d{2})/$','growse_com.blog.views.newsarchive_month'),
    (r'^projects/(?P<article_shorttitle>.+)/$','growse_com.blog.views.article'),
    (r'^misc/(?P<article_shorttitle>.+)/$','growse_com.blog.views.article'),
    (r'videos/(?P<article_shorttitle>.+)/$','growse_com.blog.views.article'),
    (r'^news/$', 'growse_com.blog.views.newsindex'),
    (r'^photos/$', 'growse_com.blog.views.photos'),
    (r'^projects/$', 'growse_com.blog.views.projectsindex'),
    (r'^videos/$', 'growse_com.blog.views.videosindex'),
    (r'^misc/$', 'growse_com.blog.views.miscindex'),
    (r'^links/$', 'growse_com.blog.views.links'),
    (r'^search/(?P<searchterm>.+)/(?P<page>\d+)/$','growse_com.blog.views.search'),
    (r'^search/(?P<searchterm>.+)/$','growse_com.blog.views.search'),
    (r'^search/$','growse_com.blog.views.search'),
    (r'^$','growse_com.blog.views.frontpage'),
)
