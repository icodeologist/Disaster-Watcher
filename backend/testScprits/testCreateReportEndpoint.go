package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type Report struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
}

var categories = []string{"flood", "animals", "landslide", "rain", "electricity outrage"}

var descriptions = map[string][]string{
	"flood": {
		"Heavy flooding on the main road, water level reaching up to knee height blocking vehicles.",
		"Flash flood hitting residential area, residents evacuating to higher ground.",
		"River overflow causing flooding in nearby barangay, roads impassable.",
	},
	"animals": {
		"A stray carabao wandering along the national highway causing traffic and danger.",
		"Pack of stray dogs blocking the road and attacking passersby.",
		"Wild boar spotted near the school premises causing panic among students.",
	},
	"landslide": {
		"Landslide blocking the highway after continuous heavy rain, debris covering two lanes.",
		"Minor landslide reported near the mountain road, rocks falling on the street.",
		"Massive landslide threatening nearby homes after 3 days of heavy rainfall.",
	},
	"rain": {
		"Continuous heavy rainfall causing poor visibility and slippery roads in the area.",
		"Severe rain causing roof damage to several houses in the subdivision.",
		"Non-stop rain for 6 hours, drainage overflowing in the market area.",
	},
	"electricity outrage": {
		"Power outage affecting the entire subdivision since last night, fallen electric post reported.",
		"Electrical fire spotted on a post near the highway, sparks flying.",
		"Entire barangay without electricity for 12 hours due to fallen transmission line.",
	},
}

var locations = []string{
	"Barangay San Jose, Manila",
	"Mountain Province, Baguio",
	"Tarlac City, Tarlac",
	"Subdivision 4, Quezon City",
	"Davao City, Davao del Sur",
	"Cebu City, Cebu",
	"Iloilo City, Iloilo",
	"Zamboanga City, Zamboanga",
	"Cagayan de Oro, Misamis Oriental",
	"General Santos City, South Cotabato",
}

var priorities = []string{"critical", "high", "medium", "low"}
var statuses = []string{"unverified", "review", "verified", "false"}

const API_URL = "http://localhost:3000/api/create"                                                                                                                                              // change this
const TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImRlbjFAZXhhbXBsZS5jb20iLCJleHAiOjE3NzIzNDYwNTksImlkIjozLCJ1c2VybmFtZSI6ImRlbjEifQ.ne1FkF-ySexMafd0jn9Rnj90rUX33CqZn3XWPPw0NtY" // change this

func main() {
	rand.Seed(time.Now().UnixNano())

	for i := 1; i <= 100; i++ {
		category := categories[rand.Intn(len(categories))]
		descs := descriptions[category]

		report := Report{
			Category:    category,
			Description: descs[rand.Intn(len(descs))],
			Location:    locations[rand.Intn(len(locations))],
			Priority:    priorities[rand.Intn(len(priorities))],
			Status:      statuses[rand.Intn(len(statuses))],
		}

		body, _ := json.Marshal(report)

		req, err := http.NewRequest("POST", API_URL, bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("[%d] Failed to create request: %v\n", i, err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+TOKEN)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[%d] Request failed: %v\n", i, err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("[%d] Category: %-20s | Priority: %-8s | Status: %-10s | Response: %d\n",
			i, report.Category, report.Priority, report.Status, resp.StatusCode)

	}
}
