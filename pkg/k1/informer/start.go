package informer

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// StartWatcher - starts watcher tooling
var logger *zap.Logger

func StartWatcher(configFile string, loggerIn *zap.Logger) error {
	logger = loggerIn
	//Setup channels
	interestingPods := make(chan Condition)
	defer close(interestingPods)
	stopper := make(chan struct{})
	defer close(stopper)

	//Process Conditions into goals
	exitScenario, _ := loadExitScenario(configFile)
	logger.Info(fmt.Sprintf("%#v", exitScenario))
	//Process Conditions into watchers
	//Start Goals tracker
	go checkPod(exitScenario, interestingPods, stopper)
	//Start Watchers
	for k, _ := range exitScenario.Conditions {
		go WatchPods(exitScenario.Conditions[k], interestingPods, stopper)
	}

	//go WatchPods(exitScenario.Conditions[1], interestingPods, stopper)
	//Check Current State - to catch events pre-informers are started

	time.Sleep(300 * time.Second)
	return nil
}

func checkPod(goal *ExitScenario, in <-chan Condition, stopper chan struct{}) {
	logger.Debug("Started Listener")
	logger.Info(fmt.Sprintf("%#v", goal))
	pendingConditions := len(goal.Conditions)

	for {
		receivedResource := <-in
		fmt.Println("\nInteresting Resource:", receivedResource)
		for key, currentCondition := range goal.Conditions {
			fmt.Println("Key:", key, "Value:", currentCondition)
			if currentCondition.Namespace == receivedResource.Namespace &&
				currentCondition.Name == receivedResource.Name &&
				currentCondition.APIVersion == receivedResource.APIVersion &&
				currentCondition.Kind == receivedResource.Kind &&
				currentCondition.Met == false {
				goal.Conditions[key].Met = true
				logger.Debug("\n Condition  Met:" + fmt.Sprintf("%#v", currentCondition))
				pendingConditions = pendingConditions - 1
				logger.Debug("\n Pending Conditions:" + fmt.Sprintf("%#v", pendingConditions))
				break
			}
		}

		fmt.Println("\n State of Conditions:", goal.Conditions)

		if pendingConditions < 1 {
			logger.Debug("All required objects found, ready to close waiting channels")
			logger.Debug(fmt.Sprintf("%#v", goal.Conditions))
			os.Exit(0)
		}
	}
}

func processExitScenario(body string) (ExitScenario, error) {
	return ExitScenario{
		Exit: 0,
		Conditions: []Condition{
			{
				Namespace:  "default",
				Name:       "sample-v3",
				Phase:      "Running",
				APIVersion: "v1",
				Kind:       "Pod",
				Met:        false,
			},
			{
				Namespace:  "default",
				Name:       "sample",
				Phase:      "Running",
				APIVersion: "v1",
				Kind:       "Pod",
				Met:        false,
			},
		},
	}, nil

}

func loadExitScenario(file string) (*ExitScenario, error) {
	exitScenario := &ExitScenario{}
	logger.Debug("Loading config file:" + file)
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		logger.Info(fmt.Sprintf("yamlFile.Get err   #%v ", err))
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, exitScenario)
	if err != nil {
		logger.Info(fmt.Sprintf("Unmarshal: %v", err))
		return nil, err
	}
	return exitScenario, nil
}
