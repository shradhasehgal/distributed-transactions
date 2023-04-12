package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"sync"

	"utils"

	"github.com/sirupsen/logrus"
)

var logrusLogger = logrus.New()

func main() {
	arguments := os.Args
	if len(arguments) != 3 {
		fmt.Printf("Please provide arguments as <client-id> <config-file>!")
		return
	}
	var wg sync.WaitGroup
	clientID := arguments[1]
	config := arguments[2]
	utils.InitlogrusLogger(logrusLogger)
	nodeToUrl := map[string]string{}
	nodeToEncoder := utils.SafeEncoderMap{M: make(map[string]*gob.Encoder)}
	logrusLogger.WithField("node", clientID).Debug("Starting node ", clientID, " with config file ", config)
	totalNodes, _ := utils.ParseConfigFile(config, nodeToUrl)
	outgoingConnectionsDone := make(chan bool, totalNodes)
	for nodeName, address := range nodeToUrl {
		wg.Add(1)
		go utils.EstablishConnection(clientID, nodeName, address, &wg, &nodeToEncoder, outgoingConnectionsDone, logrusLogger)
	}
	wg.Wait()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := scanner.Text()
		commandInfo := strings.Fields(command)
		if len(commandInfo) == 0 {
			continue
		}
		if strings.ToLower(commandInfo[0]) == "deposit" {
			if len(commandInfo) != 3 {
				continue
			}

		} else if strings.ToLower(commandInfo[0]) == "transfer" {
			if len(commandInfo) != 5 {
				continue
			}
		}
	}
}
