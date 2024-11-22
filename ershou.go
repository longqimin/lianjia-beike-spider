package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type HouseInfo struct {
	ID            string `json:"id"`
	Price         string `json:"price"`
	UnitPrice     string `json:"unit_price"`
	Room          string `json:"room"`
	HouseType     string `json:"type"`
	HouseArea     string `json:"arena"`
	CommunityName string `json:"community"`
	LocationArea  string `json:"location_area"`
	Image         string `json:"image"`
	Date          string `json:"date"`
}

func (house HouseInfo) ToCSV() string {
	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s %s %s,%s ", house.Date, house.LocationArea, house.ID, house.CommunityName, house.Price, house.UnitPrice, house.Room, house.HouseType, house.HouseArea, house.Image)
}

func merge_multiline_strings(input string, split string) string {
	line := strings.Replace(input, "\n", split, -1)
	return strings.Join(strings.Fields(line), " ")
}

func GetHouseInfo(id string) (HouseInfo, bool) {
	var cur HouseInfo
	cur.ID = id
	cur.Date = time.Now().Format("20060102")

	// Request the HTML page. https://bj.ke.com/ershoufang/101124058522.html
	addr := fmt.Sprintf("https://bj.ke.com/ershoufang/%s.html", id)
	res, err := http.Get(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return cur, false
	}
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// price := doc.Find(".sellDetailPage .overview .content .price-container")
	price := doc.Find(".price-container .price .total").Text()
	cur.Price = price + "万"

	houseInfo := doc.Find(".houseInfo")
	{
		room := houseInfo.Find(".room")
		house_type := houseInfo.Find(".type")
		area := houseInfo.Find(".area")

		mainInfo := area.Find(".mainInfo").Text()
		subinfo := area.Find(".subInfo")
		subinfo.Children().Remove()

		cur.Room = merge_multiline_strings(room.Text(), "")
		cur.HouseType = merge_multiline_strings(house_type.Text(), "")
		cur.HouseArea = merge_multiline_strings(mainInfo+" "+subinfo.Text(), "")
	}

	aroundInfo := doc.Find(".aroundInfo")
	{
		communityName := aroundInfo.Find(".communityName .info")
		areaName := aroundInfo.Find(".areaName a")
		areas := []string{}
		areaName.Each(func(i int, s *goquery.Selection) {
			if len(s.Text()) != 0 {
				areas = append(areas, s.Text())
			}
		})
		cur.CommunityName = merge_multiline_strings(communityName.Text(), "")
		cur.LocationArea = strings.Join(areas, ",")
	}

	firstpic := doc.Find(".smallpic li").First().Find("img")
	imgsrc := firstpic.AttrOr("src", "")
	if strings.Contains(imgsrc, "vrlab") {
		secondpic := doc.Find(".smallpic li:nth-child(2)").Find("img")
		imgsrc = secondpic.AttrOr("src", "")
	}
	cur.Image = imgsrc

	return cur, true

}

func load_house_ids(path string) ([]string, []string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	ids := []string{}
	var max_id string
	// Create a scanner
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text() // Get the current line
		ids = append(ids, line)
		if line > max_id {
			max_id = line
		}

	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	guess_ids := []string{}
	max, _ := strconv.ParseInt(max_id, 10, 64)
	for i := int64(0); i < 100000; i++ {
		guess_ids = append(guess_ids, fmt.Sprintf("%d", max+i))
	}
	return ids, guess_ids
}

func main() {
	// house, _ := GetHouseInfo("101124058522")
	// fmt.Printf("%s\n", house.ToCSV())
	// return
	houseIds, guess_ids := load_house_ids("./bj.ke.ids")
	houseIds = append(houseIds, guess_ids...)

	parallel := 16
	var wg sync.WaitGroup
	wg.Add(parallel)

	worker := func(partition int) {
		defer wg.Done()
		for id, houseID := range houseIds {
			if id%parallel == partition {
				house, valid := GetHouseInfo(houseID)
				if valid {
					fmt.Printf("%s\n", house.ToCSV())
				}
			}
		}
	}

	for i := 0; i < parallel; i++ {
		go worker(i)
	}

	wg.Wait()

}
