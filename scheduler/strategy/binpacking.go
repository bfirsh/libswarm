package strategy

import (
	"errors"
	"sort"
	"strconv"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

var (
	ErrNoResourcesAvailable = errors.New("no resources available to schedule container")
)

type BinPackingPlacementStrategy struct {
	ratio int64
}

func (p *BinPackingPlacementStrategy) Initialize(opts string) error {
	overcommitRatio, err := strconv.ParseFloat(opts, 64)
	p.ratio = int64(overcommitRatio * 100)
	return err
}

func (p *BinPackingPlacementStrategy) PlaceContainer(config *dockerclient.ContainerConfig, nodes []*cluster.Node) (*cluster.Node, error) {
	scores := scores{}

	for _, node := range nodes {
		nodeMemory := node.Memory + (node.Memory * p.ratio / 100)
		nodeCpus := node.Cpus + (node.Cpus * p.ratio / 100)

		// Skip nodes that are smaller than the requested resources.
		if nodeMemory < int64(config.Memory) || nodeCpus < config.CpuShares {
			continue
		}

		var (
			cpuScore    int64 = 100
			memoryScore int64 = 100
		)

		if config.CpuShares > 0 {
			cpuScore = (node.ReservedCpus() + config.CpuShares) * 100 / nodeCpus
		}
		if config.Memory > 0 {
			memoryScore = (node.ReservedMemory() + config.Memory) * 100 / nodeMemory
		}
		var total = ((cpuScore + memoryScore) / 200) * 100

		if cpuScore <= 100 && memoryScore <= 100 {
			scores = append(scores, &score{node: node, score: total})
		}
	}

	if len(scores) == 0 {
		return nil, ErrNoResourcesAvailable
	}

	sort.Sort(scores)

	return scores[0].node, nil
}

type score struct {
	node  *cluster.Node
	score int64
}

type scores []*score

func (s scores) Len() int {
	return len(s)
}

func (s scores) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s scores) Less(i, j int) bool {
	var (
		ip = s[i]
		jp = s[j]
	)

	return ip.score > jp.score
}
