package internal

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerImageBasicInfo struct {
	// Basic Image Properties
	ImageID   string   `json:"image_id"`   // Showing first 12 chars in UI
	RepoTags  []string `json:"repo_tags"`  // Main repo:tag info
	Size      string   `json:"size"`       // Formatted human-readable size
	CreatedAt string   `json:"created_at"` // For calculating age
}

type DockerImageDetailInfo struct {
	// Additional details for the expanded view
	ParentId     string            `json:"parent_id,omitempty"`
	RepoDigests  []string          `json:"repo_digests,omitempty"`
	Architecture string            `json:"architecture,omitempty"`
	Os           string            `json:"os,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`

	// Added field for container usage count
	ContainerCount int `json:"container_count"`
}

type DockerImage struct {
	BasicInfo  DockerImageBasicInfo  `json:"basic_info"`
	DetailInfo DockerImageDetailInfo `json:"detail_info,omitempty"`
}

type DockerContainerInfo struct {
	// Basic identification
	ID    string `json:"id"`
	Names string `json:"names"`
	Image string `json:"image"`

	// Status information
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	HealthCheck string `json:"health,omitempty"`

	// Basic network and config
	Ports         []PortInfo `json:"ports,omitempty"`
	RestartPolicy string     `json:"restart_policy"`
}

type PortInfo struct {
	PrivatePort string `json:"private_port"` // Port inside container
	PublicPort  string `json:"public_port"`  // Port on host
	Protocol    string `json:"protocol"`     // tcp, udp
	HostIP      string `json:"host_ip"`      // 0.0.0.0, 127.0.0.1, etc.
}

type InspectInfo []container.InspectResponse

type DockerService interface {
	GetImages() ([]DockerImage, error)
	GetContainersInfo() ([]DockerContainerInfo, error)
}

type DockerManager struct {
	dockerClient *client.Client
}

func NewDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerManager{
		dockerClient: cli,
	}, nil
}

func MapExposedPort(portMap nat.PortMap) []PortInfo {
	var portInfos []PortInfo
	for port, binding := range portMap {
		if len(binding) > 0 {
			portInfo := PortInfo{
				PrivatePort: port.Port(),
				PublicPort:  binding[0].HostPort,
				HostIP:      binding[0].HostIP,
				Protocol:    port.Proto(),
			}
			portInfos = append(portInfos, portInfo)
		}
	}
	return portInfos
}

// TODO:: Implement container usage
func GetConainerUsage(id string) (no int) { return 0 }

func ParseUTCTime(timestamp string) string {
	time, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		log.Fatal(err.Error())
	}
	return time.Format("2006-01-02 3:4:5 PM")
}

func (d *DockerManager) GetImages() ([]DockerImage, error) {
	var dockerImages []DockerImage
	images, err := d.dockerClient.ImageList(context.Background(), image.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, img := range images {
		createdTime := time.Unix(img.Created, 0).Format("Jan 2, 2006 3:04 PM")

		//Get basic image information
		currentImageBasicInfo := DockerImageBasicInfo{
			ImageID:   strings.TrimPrefix(img.ID, "sha256:")[:8],
			RepoTags:  img.RepoTags,
			Size:      fmt.Sprintf("%.2fMB", float64(img.Size)/float64(1024*1024)),
			CreatedAt: createdTime,
		}

		// Get detailed image information
		inspect, err := d.dockerClient.ImageInspect(context.Background(), img.ID)
		if err != nil {
			continue
		}

		currentImageDetailInfo := DockerImageDetailInfo{
			ParentId:       inspect.Parent,
			Architecture:   inspect.Architecture,
			Os:             inspect.Os,
			ContainerCount: int(img.Containers),
			Labels:         inspect.Config.Labels,
		}
		dockerImages = append(dockerImages, DockerImage{BasicInfo: currentImageBasicInfo, DetailInfo: currentImageDetailInfo})
	}
	return dockerImages, nil
}

func (d *DockerManager) GetContainersInfo() ([]DockerContainerInfo, error) {

	var containerDetails []DockerContainerInfo
	basicInfo, err := d.dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, c := range basicInfo {
		inspectInfo, err := d.dockerClient.ContainerInspect(context.Background(), c.ID)
		if err != nil {
			return nil, err
		}

		//KNOWLEDGE: ExposedPort does not mean these are the posts actually exposed to HOST
		//That can be found in NetworkSettings.Ports
		containerInfo := DockerContainerInfo{
			ID:            inspectInfo.ID[:12],
			Names:         inspectInfo.Name,
			Image:         inspectInfo.Config.Image,
			Status:        inspectInfo.State.Status,
			CreatedAt:     ParseUTCTime(inspectInfo.Created),
			Ports:         MapExposedPort(inspectInfo.NetworkSettings.Ports),
			RestartPolicy: string(inspectInfo.HostConfig.RestartPolicy.Name),
		}
		containerDetails = append(containerDetails, containerInfo)
	}
	return containerDetails, nil
}
