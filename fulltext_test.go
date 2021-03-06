package main

import (
	"github.com/blevesearch/bleve"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestAddingFilesToIndexAddsTheFilesToTheIndex(t *testing.T) {
	indexTempDir, err := ioutil.TempDir("", "testindex")
	defer os.RemoveAll(indexTempDir)
	assert.Nil(t, err)
	t.Logf("Index Tempdir: %v", indexTempDir)

	tempDir, err := ioutil.TempDir("", "testprefix")
	defer os.RemoveAll(tempDir)
	assert.Nil(t, err)
	t.Logf("Posts Tempdir: %v", tempDir)

	firstPostDir := filepath.Join(tempDir, "post1", "_posts")
	err = os.MkdirAll(firstPostDir, os.ModePerm)
	assert.Nil(t, err)
	t.Logf("Created %v", firstPostDir)

	secondPostDir := filepath.Join(tempDir, "post2", "_posts")
	err = os.MkdirAll(secondPostDir, os.ModePerm)
	assert.Nil(t, err)
	t.Logf("Created %v", secondPostDir)

	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(firstPostDir, "first.md"))
	assert.Nil(t, err)
	_, err = os.Create(filepath.Join(secondPostDir, "second.md"))
	assert.Nil(t, err)

	mapping := bleve.NewIndexMapping()
	index, err := bleve.New(indexTempDir, mapping)
	assert.Nil(t, err)

	err = addFilesToIndex(tempDir, index)
	assert.Nil(t, err)

	count, err := index.DocCount()
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), count)
}

func TestAddSingleFileToIndexAddsTheFileToTheIndex(t *testing.T) {
	indexTempDir, err := ioutil.TempDir("", "testindex")
	defer os.RemoveAll(indexTempDir)
	assert.Nil(t, err)
	t.Logf("Index Tempdir: %v", indexTempDir)

	tempDir, err := ioutil.TempDir("", "testprefix")
	defer os.RemoveAll(tempDir)
	assert.Nil(t, err)
	t.Logf("Posts Tempdir: %v", tempDir)

	tempFile, err := ioutil.TempFile(tempDir, "testprefix1")
	assert.Nil(t, err)

	mapping := bleve.NewIndexMapping()
	index, err := bleve.New(indexTempDir, mapping)
	if err != nil {
		panic(err)
	}
	assert.Nil(t, err)

	err = addFileToIndex(tempFile.Name(), index)
	assert.Nil(t, err)
	count, err := index.DocCount()
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), count)
}

func TestSearchingIndexForFileContentReturnsResult(t *testing.T) {
	indexTempDir, err := ioutil.TempDir("", "testindex")
	defer os.RemoveAll(indexTempDir)
	assert.Nil(t, err)
	t.Logf("Index Tempdir: %v", indexTempDir)

	content := "title:content\n---\ncontent"
	tempDir, err := ioutil.TempDir("", "testprefix")
	defer os.RemoveAll(tempDir)
	assert.Nil(t, err)

	tempFile, err := ioutil.TempFile(tempDir, "testprefix1")
	assert.Nil(t, err)

	ioutil.WriteFile(tempFile.Name(), []byte(content), os.ModePerm)

	mapping := bleve.NewIndexMapping()
	index, err := bleve.New(indexTempDir, mapping)
	assert.Nil(t, err)

	err = addFileToIndex(tempFile.Name(), index)
	assert.Nil(t, err)
	count, err := index.DocCount()
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), count)

	searchForm := SearchForm{SearchTerm: "content"}
	searchResult, err := searchIndexForThings(index, searchForm)

	assert.Nil(t, err)
	assert.Equal(t, 0, searchResult.Status.Failed)
	assert.Equal(t, 1, searchResult.Status.Successful)
	assert.Equal(t, 0, len(searchResult.Status.Errors))
	assert.Equal(t, 1, len(searchResult.Hits))
}

func TestSearchingIndexForNonFileContentReturnsResult(t *testing.T) {
	indexTempDir, err := ioutil.TempDir("", "testindex")
	defer os.RemoveAll(indexTempDir)
	assert.Nil(t, err)
	t.Logf("Index Tempdir: %v", indexTempDir)

	content := "content"
	tempDir, err := ioutil.TempDir("", "testprefix")
	defer os.RemoveAll(tempDir)
	assert.Nil(t, err)

	tempFile, err := ioutil.TempFile(tempDir, "testprefix1")
	assert.Nil(t, err)

	ioutil.WriteFile(tempFile.Name(), []byte(content), os.ModePerm)

	mapping := bleve.NewIndexMapping()
	index, err := bleve.New(indexTempDir, mapping)
	assert.Nil(t, err)

	err = addFileToIndex(tempFile.Name(), index)
	assert.Nil(t, err)
	count, err := index.DocCount()
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), count)

	query := bleve.NewMatchQuery("not")
	searchRequest := bleve.NewSearchRequest(query)
	searchResult, err := index.Search(searchRequest)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), searchResult.Total)
}
