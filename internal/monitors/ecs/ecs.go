// Package ecs contains a monitor for getting metrics about containers running
// in a docker engine.
package ecs

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"context"
	"time"
	"fmt"

	dtypes "github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/common/ecs"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	dmonitor "github.com/signalfx/signalfx-agent/internal/monitors/docker"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	log "github.com/sirupsen/logrus"
)

const monitorType = "ecs-metadata"

// MONITOR(ecs-metadata): This monitor reads container stats from a
// [ECS Task Metadata Endpoint version 2](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v2.html). 
// 
// This currently does not support CPU share/quota metrics.

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`

	// The URL of the ECS task metadata. Default is http://169.254.170.2/metadata, which is hardcoded by AWS for version 2.
	MetadataEndpoint string `yaml:"metadataEndpoint" default:"http://169.254.170.2/metadata"`
	// The URL of the ECS container stats. Default is http://169.254.170.2/stats, which is hardcoded by AWS for version 2.
	StatsEndpoint string `yaml:"statsEndpoint" default:"http://169.254.170.2/stats"`
	// The maximum amount of time to wait for API requests
	TimeoutSeconds int `yaml:"timeoutSeconds" default:"5"`
	// A mapping of container label names to dimension names. The corresponding
	// label values will become the dimension value for the mapped name.  E.g.
	// `io.kubernetes.container.name: container_spec_name` would result in a
	// dimension called `container_spec_name` that has the value of the
	// `io.kubernetes.container.name` container label.
	LabelsToDimensions map[string]string `yaml:"labelsToDimensions"`
	// A list of filters of images to exclude.  Supports literals, globs, and
	// regex.
	ExcludedImages []string `yaml:"excludedImages"`
}

// Monitor for ECS Metadata
type Monitor struct {
	Output  types.Output
	cancel  func()
	client  *http.Client
	conf *Config
	ctx context.Context
	timeout time.Duration
	taskDimensions map[string]string
	containers map[string]ecs.Container
	// key : container docker id, tells if stats for the container should be ignored.
	// Usually the container was filtered out by excludedImages
	// or container metadata is not received.
	shouldIgnore map[string]bool
	imageFilter filter.StringFilter
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	var err error
	m.imageFilter, err = filter.NewBasicStringFilter(conf.ExcludedImages)
	if err != nil {
		return errors.Wrapf(err, "Could not load excluded image filter")
	}

	m.conf = conf
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = &http.Client{
		Timeout: m.timeout,
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())

	isRegistered := false

	utils.RunOnInterval(m.ctx, func() {
		if !isRegistered {
			m.containers = map[string]ecs.Container{}
			m.shouldIgnore = map[string]bool{}
			m.fetchContainers()
			
			isRegistered = true
		}

		m.fetchStatsForAll()

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Fetch all containers in the task and load it to the monitor
func (m *Monitor) fetchContainers() {
	body, err := getMetadata(m.client, m.conf.MetadataEndpoint)
	if err != nil {
		logger.WithError(err).Error("Failed to read ECS container data - %v", err)
		return
	}

	task := ecs.TaskMetadata{}
	if err := json.Unmarshal(body, &task); err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
		}).Error("Could not parse ecs metadata json")
		return
	}
	m.taskDimensions = task.GetDimensions()

	for i := range task.Containers {
		if (m.imageFilter == nil ||
			!m.imageFilter.Matches(task.Containers[i].Image)) &&
			 // Containers that are specified in the task definition are of type NORMAL. This will filter out all AWS internal containers
			task.Containers[i].Type == "NORMAL" {
			m.containers[task.Containers[i].DockerId] = task.Containers[i]
			m.shouldIgnore[task.Containers[i].DockerId] = false
		} else {
			m.shouldIgnore[task.Containers[i].DockerId] = true
		}
	}
}

// Fetch a container with given container docker ID and load it to the monitor
// If the container is successfully received, return true. Else, return false
func (m *Monitor) fetchContainer(dockerId string) bool {
	body, err := getMetadata(m.client, fmt.Sprintf("%s/%s", m.conf.MetadataEndpoint, dockerId))

	if err != nil {
		logger.WithError(err).Error("Failed to read ECS container data - %v", err)
		return false
	}

	var container ecs.Container

	if err := json.Unmarshal(body, &container); err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
		}).Error("Could not parse ecs container json")
		return false
	}

	if (m.imageFilter != nil && m.imageFilter.Matches(container.Image)) &&
		container.Type == "NORMAL" {
		return false
	}

	m.containers[dockerId] = container
	return true
}

func (m *Monitor) fetchStatsForAll() {
	_, cancel := context.WithTimeout(m.ctx, m.timeout)
	
	body, err := getMetadata(m.client, m.conf.StatsEndpoint)

	if err != nil {
		cancel()
		logger.WithError(err).Error("Failed to read ECS stats - %v", err)
		return
	}

	var stats map[string]dtypes.StatsJSON

	if err := json.Unmarshal(body, &stats); err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
		}).Error("Could not parse stats json")
		return
	}

	for dockerId := range stats {
		if m.shouldIgnore[dockerId] {
			continue
		}
		
		container, ok := m.containers[dockerId]
		if !ok {
			logger.Debugf("Container not found for id %s. Fetching...", dockerId)
			if received := m.fetchContainer(dockerId); !received {
				continue
			}
			container = m.containers[dockerId]
		}

		containerJSON := &dtypes.ContainerJSON{
			ContainerJSONBase: &dtypes.ContainerJSONBase {
				ID: dockerId,
				Name: container.Name,
			},
			Config: &dcontainer.Config{
				Image: 		container.Image,
				Hostname: 	container.Networks[0].IPAddresses[0],
			},
		}
	
		containerStat := stats[dockerId]
		dps, err := dmonitor.ConvertStatsToMetrics(containerJSON, &containerStat)
		cancel()
	
		if err != nil {
			logger.WithError(err).Errorf("Could not convert docker stats for container id %s", dockerId)
			return
		}
	
		for i := range dps {
			logger.WithFields(log.Fields{
				"dims": m.taskDimensions,
			}).Debugf("Dimensions!!!!!!")
			// Add task metadata to dimensions
			for dimName, v := range m.taskDimensions {
				dps[i].Dimensions[dimName] = v
			}

			for k, dimName := range m.conf.LabelsToDimensions {
				if v := m.containers[dockerId].Labels[k]; v != "" {
					dps[i].Dimensions[dimName] = v
				}
			}
			
			m.Output.SendDatapoint(dps[i])
		}
	}
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func getMetadata(client *http.Client, endpoint string) ([]byte, error) {
	response, err := client.Get(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not connect to %s", endpoint)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Could not connect to %s : %s ", endpoint, http.StatusText(response.StatusCode)))
	}

	body, err := ioutil.ReadAll(response.Body)
	return body, err
}
