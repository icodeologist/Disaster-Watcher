package models

type GeocodingResult struct {
	Latitude     string `json:"lat"`
	Longitude    string `json:"lon"`
	LocationName string `json:"locaiton_name"`
}
