package models

type GeocodingResult struct {
	Latitude     string `json:"lat"`
	Longitude    string `json:"lon"`
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	LocationName string `json:"locaiton_name"`
}
