package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Location struct {
	Latitude  string `json:"lat"`
	Longitude string `json:"lon"`
}

// using nominatim api to for forward geocoding
// means we give a place name and it returns a json data
func ForwardGeo(location string) []Location {
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%v&format=json", location)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	var locs []Location

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	// json data to go data type
	err = json.Unmarshal(body, &locs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(locs)
	return locs
}
