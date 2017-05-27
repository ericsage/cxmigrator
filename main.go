package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

//NetworkSearchResponse is the JSON format returned by the NDEx network search endpoint.
type NetworkSearchResponse struct {
	NumFound int              `json:"numFound"`
	Networks []NetworkSummary `json:"networks"`
}

//NetworkSummary describes a single Network in NDEx.
type NetworkSummary struct {
	Name       string
	ExternalID string `json:"externalId"`
	EdgeCount  int    `json:"edgeCount"`
	NodeCount  int    `json:"nodeCount"`
	/*
		Version          string        `json:"version"`
		ModificationTime int           `json:"modificationTime"`
		CreationTime     int           `json:"creationTime"`
		Owner            string        `json:"owner"`
		OwnerUUID        string        `json:"ownerUUID"`
		IsReadyOnly      bool          `json:"isReadOnly"`
		IsValid          bool          `json:"isValid"`
		IsShowcase       bool          `json:"isShowcase"`
		IsDeleted        bool          `json:"isDeleted"`
		SubnetworkIds    []string      `json:"subnetworkIds"`
		Properties       []interface{} `json:"properties"`
		Warnings         []interface{} `json:"warnings"`
		ErrorMessage     string        `json:"errorMessage"`
		Visibility       string        `json:"visibility"`
		URI              string        `json:"uri"`
		Description      string        `json:"description"`
	*/
}

// NetworkDescriptor is a manifest of an NDEx network that will be saved to a file.
type NetworkDescriptor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NodeCount int    `json:"nodeCount"`
	EdgeCount int    `json:"edgeCount"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func createManifest(manifestFile string) {
	body := strings.NewReader("{ \"searchString\": \"\" }")
	resp, err := http.Post("http://ndexbio.org/v2/search/network?size=1000000", "application/json", body)
	check(err)
	var nsp NetworkSearchResponse
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&nsp)
	networks := []NetworkDescriptor{}
	if nsp.NumFound != len(networks) {
		panic("Number of networks in memory not equal to the number found by the search API.")
	}
	fmt.Println("Number of networks that will be written to the manifest:", nsp.NumFound)
	for _, net := range nsp.Networks {
		nd := NetworkDescriptor{
			net.ExternalID,
			net.Name,
			net.NodeCount,
			net.EdgeCount,
		}
		networks = append(networks, nd)
	}
	file, err := os.Create(manifestFile)
	check(err)
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	enc.Encode(networks)
}

func readManifest(manifestFile string) []NetworkDescriptor {
	file, err := os.Open(manifestFile)
	check(err)
	dec := json.NewDecoder(file)
	var nds []NetworkDescriptor
	dec.Decode(&nds)
	return nds
}

func downloadNetwork(id string) io.Reader {
	resp, err := http.Get("http://ndexbio.org/v2/network/" + id)
	check(err)
	return resp.Body
}

func transfer(uploader *s3manager.Uploader, bucketName string, name string, id string, index int, wg *sync.WaitGroup) {
	netData := downloadNetwork(id)
	upParams := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &id,
		Body:   netData,
	}
	fmt.Println("Upload ", "num:", index, "name:", name, "id:", id)
	_, err := uploader.Upload(upParams)
	check(err)
	wg.Done()
}

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	uploader := s3manager.NewUploader(sess)
	bucketName := "cx.ndex.test"
	networks := readManifest("./networks.json")
	var wg sync.WaitGroup
	for index, network := range networks {
		wg.Add(1)
		go transfer(uploader, bucketName, network.Name, network.ID, index, &wg)
	}
	wg.Wait()
}
