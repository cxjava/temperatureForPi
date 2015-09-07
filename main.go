package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pmylund/go-cache"
)

var (
	cpu     *cache.Cache
	saveSig = make(chan os.Signal)
	file    = filepath.Join(path.Dir(os.Args[0]), "cache.dat")
)

func init() {
	readConfig()

	file = filepath.Join(path.Dir(os.Args[0]), config.CacheFileName)
	cpu = cache.New(time.Second*time.Duration(60*60*config.CacheExpire), time.Second*time.Duration(config.CleanupInterval))
	cpu.LoadFile(file)
	saveCacheToFile()
	fetchTemperature()

	signal.Notify(saveSig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
}

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("views/*")
	r.Static("/assets", "./assets")
	r.GET("/", func(c *gin.Context) {
		s := getAllTemperatures()
		sort.Sort(s)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"data": s,
		})
	})
	r.GET("/refresh", func(c *gin.Context) {
		SaveCPUTemperature()
		c.Redirect(http.StatusFound, "/")
	})
	r.Run(config.ListenAddress)
}

func getAllTemperatures() (ps Points) {
	its := cpu.Items()
	ps = make(Points, 0, len(its))
	for k, v := range its {
		ps = append(ps, *&Point{
			X: k,
			Y: v.Object.(float64),
		})
	}
	return
}

type Point struct {
	X string
	Y float64
}

type Points []Point

func (a Points) Len() int           { return len(a) }
func (a Points) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Points) Less(i, j int) bool { return a[i].X < a[j].X }

func SaveCPUTemperature() {
	t := getCPUTemperature()
	cpu.Set(time.Now().Local().Format(config.DataFormatStyle), t, cache.DefaultExpiration)
}

func getCPUTemperature() (t float64) {
	data, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		fmt.Println("ioutil.ReadFile err:", err)
		return
	}
	tStr := strings.Replace(string(data), "\n", "", -1)
	t, err = strconv.ParseFloat(tStr, 10)
	if err != nil {
		fmt.Println("getCPUTemperature strconv.ParseFloat err:", err)
		return
	}
	t = t / 1000
	return
}

func getGPUTemperature() (t float64) {
	cmd := exec.Command("/opt/vc/bin/vcgencmd", "measure_temp")
	data, err := cmd.Output() //return  temp=41.2'C
	if err != nil {
		fmt.Println("cmd.Output err:", err)
		return
	}
	tStr := strings.Replace(string(data), `temp=`, "", -1)
	tStr = strings.Replace(tStr, "\n", "", -1)
	tStr = strings.Replace(tStr, `'C`, "", -1)
	t, err = strconv.ParseFloat(tStr, 10)
	if err != nil {
		fmt.Println("getGPUTemperature strconv.ParseFloat err:", err)
		return
	}
	return
}

func saveCacheToFile() {
	go func() {
		hb := time.Tick(time.Duration(config.SaveFileInterval) * time.Second)
		for {
			select {
			case sig := <-saveSig:
				cpu.SaveFile(file)
				switch sig {
				case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
					os.Exit(0)
				case syscall.SIGHUP:
					log.Println("Clear cache!")
					cpu.Flush()
				}
			case <-hb:
				cpu.SaveFile(file)
			}
		}
	}()
}

func fetchTemperature() {
	go func() {
		hb := time.Tick(time.Duration(config.GetInterval) * time.Minute)
		for {
			select {

			case <-hb:
				SaveCPUTemperature()
			}
		}
	}()
}
