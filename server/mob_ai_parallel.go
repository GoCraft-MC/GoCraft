package server

import (
	"runtime"
	"sync"
	"sync/atomic"

	corentity "GoCraft/core/entity"
)

const maximumPassiveAIWorkers = 8

// tickPassiveAIParallel runs only the per-entity passive AI phase. Candidate
// selection and mobAI map creation remain on the tick goroutine; workers then
// read the map and mutate distinct entity/AI pairs. Shared effects such as
// metadata broadcasts are returned and applied by the tick goroutine.
func (s *Server) tickPassiveAIParallel(entities []*corentity.Entity, players []naturalSpawnPlayer) []*corentity.Entity {
	candidates := make([]*corentity.Entity, 0, len(entities))
	for _, entity := range entities {
		if entity.Dead || !isPassiveMob(entity.Type) || !entityWithinSimulationRange(entity, players, 128) {
			continue
		}
		ai := s.mobAIFor(entity)
		if entity.Type == corentity.TypeVillager {
			s.tickVillagerBedClaim(entity, ai)
		}
		candidates = append(candidates, entity)
	}
	workerCount := passiveAIWorkerCount(len(candidates), runtime.GOMAXPROCS(0))
	if workerCount == 0 {
		return nil
	}

	woke := make([]bool, len(candidates))
	if workerCount == 1 {
		for index, entity := range candidates {
			woke[index] = s.tickPassiveMobAI(entity) && entity.Type == corentity.TypeVillager
		}
	} else {
		var next atomic.Int64
		var workers sync.WaitGroup
		workers.Add(workerCount)
		for range workerCount {
			go func() {
				defer workers.Done()
				for {
					index := int(next.Add(1)) - 1
					if index >= len(candidates) {
						return
					}
					entity := candidates[index]
					woke[index] = s.tickPassiveMobAI(entity) && entity.Type == corentity.TypeVillager
				}
			}()
		}
		workers.Wait()
	}

	villagers := make([]*corentity.Entity, 0)
	for index, changed := range woke {
		if changed {
			villagers = append(villagers, candidates[index])
		}
	}
	return villagers
}

func passiveAIWorkerCount(entityCount, availableProcessors int) int {
	if entityCount <= 0 {
		return 0
	}
	if availableProcessors < 1 {
		availableProcessors = 1
	}
	if availableProcessors > maximumPassiveAIWorkers {
		availableProcessors = maximumPassiveAIWorkers
	}
	if availableProcessors > entityCount {
		availableProcessors = entityCount
	}
	return availableProcessors
}
