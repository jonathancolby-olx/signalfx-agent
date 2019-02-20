package ecs

import "strings"

type TaskMetadata struct {
	ClusterName	string `json:"Cluster"`
	TaskARN		string `json:"TaskARN"`
	Family		string `json:"Family"`
	Revision	string `json:"Revision"`
	KnownStatus	string `json:"KnownStatus"`
	Containers  []Container `json:"Containers"`
}

func (task *TaskMetadata) GetDimensions() map[string]string {
	dims := make(map[string]string, 0)
	if idx := strings.Index(task.ClusterName, "/"); idx >= 0 {
		dims["ClusterName"] = task.ClusterName[idx+1: len(task.ClusterName)]
	} else {
		dims["ClusterName"] = task.ClusterName
	}
	dims["TaskARN"] = task.TaskARN
	dims["Family"] = task.Family
	dims["Revision"] = task.Revision
	dims["KnownStatus"] = task.KnownStatus

	return dims
}

type Container struct {
	DockerId 	string `json:"DockerId"`
	Name 		string `json:"DockerName"`
	Image 		string `json:"Image"`
	KnownStatus string `json:"KnownStatus"`
	Type		string `json:"Type"`
	Labels 		map[string]string `json:"Labels"`
	Networks 	[]struct {
		NetworkMode	string `json:"NetworkMode"`
		IPAddresses	[]string `json:"IPv4Addresses"`
	}
}
