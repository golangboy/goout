package goout

import (
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"io"
	"net/http"
	"os"
)

func QueryIp(Ipv4 string) string {
	var dbPath = "ip2region.xdb"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		xdbUrl := "https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.xdb"
		get, err := http.Get(xdbUrl)
		if err != nil {
			return ""
		}
		defer get.Body.Close()
		create, err := os.Create(dbPath)
		if err != nil {
			return ""
		}
		defer create.Close()
		io.Copy(create, get.Body)
	}
	searcher, _ := xdb.NewWithFileOnly(dbPath)
	region, err := searcher.SearchByStr(Ipv4)
	if err != nil {
		return ""
	}
	return region
}
