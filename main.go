package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/pokidovea/mimicro/mockServer"
	"github.com/pokidovea/mimicro/statistics"
)

func checkConfig(configPath string) error {
	err := mockServer.CheckConfig(configPath)

	if err == nil {
		fmt.Println("Config is valid")
		return nil
	}

	fmt.Printf("Config is not valid. See errors below: \n %s \n", err.Error())
	return err
}

func main() {

	configPath := flag.String("config", "", "a path to configuration file")
	checkConf := flag.Bool("check", false, "validates passed config")
	flag.Parse()

	err := checkConfig(*configPath)

	if err != nil {
		os.Exit(1)
	}

	if *checkConf == true {
		os.Exit(0)
	}

	serverCollection, err := mockServer.Load(*configPath)

	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var statisticsChannel chan statistics.Request

	if serverCollection.CollectStatistics {
		wg.Add(1)

		statisticsChannel = make(chan statistics.Request)

		statisticsCollector := statistics.Collector{Chan: statisticsChannel}
		go statisticsCollector.Run(&wg)
	}

	for _, server := range serverCollection.Servers {
		wg.Add(1)
		go server.Serve(statisticsChannel, &wg)
	}

	wg.Wait()
	log.Printf("Mimicro successfully down")
}
