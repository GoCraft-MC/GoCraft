// Package navigation implements edition-neutral ground-mob navigation.
package navigation

import (
	"container/heap"
	"math"

	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

const (
	maximumStepUp = 1
	maximumFall   = 3
)

type nodeKey struct{ x, y, z int }

type standableResult struct {
	y  int
	ok bool
}

// Adjacent A* nodes test the same columns repeatedly. Cache each result only
// for this search, so the next search observes terrain edits.
type walkEvaluator struct {
	world *coreworld.World
	cache map[nodeKey]standableResult
}

func (e *walkEvaluator) standableY(x, y, z int) (int, bool) {
	key := nodeKey{x: x, y: y, z: z}
	if result, found := e.cache[key]; found {
		return result.y, result.ok
	}
	standY, ok := standableY(e.world, x, y, z)
	e.cache[key] = standableResult{y: standY, ok: ok}
	return standY, ok
}

type searchNode struct {
	key       nodeKey
	g, h      float64
	parent    *searchNode
	heapIndex int
	closed    bool
}

func (n *searchNode) score() float64 { return n.g + n.h }

type openHeap []*searchNode

func (h openHeap) Len() int { return len(h) }
func (h openHeap) Less(i, j int) bool {
	if h[i].score() == h[j].score() {
		return h[i].h < h[j].h
	}
	return h[i].score() < h[j].score()
}
func (h openHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex, h[j].heapIndex = i, j
}
func (h *openHeap) Push(value any) {
	node := value.(*searchNode)
	node.heapIndex = len(*h)
	*h = append(*h, node)
}
func (h *openHeap) Pop() any {
	old := *h
	node := old[len(old)-1]
	node.heapIndex = -1
	*h = old[:len(old)-1]
	return node
}

// FindPath runs a bounded A* walk-node search based on Pumpkin's
// WalkNodeEvaluator/Navigator. The returned positions are block-centred feet
// positions and omit the starting node. reached is false when the best partial
// path is returned because the exact goal could not be occupied.
func FindPath(world *coreworld.World, start, goal spatial.Vec3, maxVisited int) ([]spatial.Vec3, bool) {
	if world == nil || maxVisited <= 0 {
		return nil, false
	}
	evaluator := &walkEvaluator{world: world, cache: make(map[nodeKey]standableResult)}
	startKey, ok := nearestStandable(evaluator, int(math.Floor(start.X)), int(math.Floor(start.Y)), int(math.Floor(start.Z)))
	if !ok {
		return nil, false
	}
	goalKey, goalStandable := nearestStandable(evaluator, int(math.Floor(goal.X)), int(math.Floor(goal.Y)), int(math.Floor(goal.Z)))
	if !goalStandable {
		goalKey = nodeKey{x: int(math.Floor(goal.X)), y: int(math.Floor(goal.Y)), z: int(math.Floor(goal.Z))}
	}

	nodes := make(map[nodeKey]*searchNode, min(maxVisited, 128))
	startNode := &searchNode{key: startKey, h: heuristic(startKey, goalKey), heapIndex: -1}
	nodes[startKey] = startNode
	open := openHeap{startNode}
	heap.Init(&open)
	best := startNode

	for open.Len() > 0 && len(nodes) <= maxVisited {
		current := heap.Pop(&open).(*searchNode)
		if current.closed {
			continue
		}
		current.closed = true
		if current.h < best.h {
			best = current
		}
		if current.key == goalKey && goalStandable {
			return reconstruct(current), true
		}

		for _, candidate := range neighbours(evaluator, current.key) {
			stepCost := movementCost(current.key, candidate)
			newCost := current.g + stepCost
			node, exists := nodes[candidate]
			if !exists {
				node = &searchNode{key: candidate, g: math.Inf(1), h: heuristic(candidate, goalKey), heapIndex: -1}
				nodes[candidate] = node
			}
			if node.closed || newCost >= node.g {
				continue
			}
			node.g = newCost
			node.parent = current
			if node.heapIndex >= 0 {
				heap.Fix(&open, node.heapIndex)
			} else {
				heap.Push(&open, node)
			}
		}
	}
	if best == startNode {
		return nil, false
	}
	return reconstruct(best), false
}

func nearestStandable(evaluator *walkEvaluator, x, y, z int) (nodeKey, bool) {
	for radius := 0; radius <= 2; radius++ {
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if radius > 0 && abs(dx) != radius && abs(dz) != radius {
					continue
				}
				if standY, ok := evaluator.standableY(x+dx, y, z+dz); ok {
					return nodeKey{x: x + dx, y: standY, z: z + dz}, true
				}
			}
		}
	}
	return nodeKey{}, false
}

func neighbours(evaluator *walkEvaluator, current nodeKey) []nodeKey {
	cardinal := make(map[[2]int]nodeKey, 4)
	result := make([]nodeKey, 0, 8)
	for _, offset := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if y, ok := evaluator.standableY(current.x+offset[0], current.y, current.z+offset[1]); ok {
			next := nodeKey{x: current.x + offset[0], y: y, z: current.z + offset[1]}
			cardinal[offset] = next
			result = append(result, next)
		}
	}
	for _, offset := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		first, firstOK := cardinal[[2]int{offset[0], 0}]
		second, secondOK := cardinal[[2]int{0, offset[1]}]
		if !firstOK || !secondOK || abs(first.y-current.y) > maximumStepUp || abs(second.y-current.y) > maximumStepUp {
			continue
		}
		y, ok := evaluator.standableY(current.x+offset[0], current.y, current.z+offset[1])
		if ok {
			result = append(result, nodeKey{x: current.x + offset[0], y: y, z: current.z + offset[1]})
		}
	}
	return result
}

func standableY(world *coreworld.World, x, referenceY, z int) (int, bool) {
	candidates := [maximumStepUp + maximumFall + 1]int{referenceY, referenceY + 1, referenceY - 1, referenceY - 2, referenceY - 3}
	for _, y := range candidates {
		if y < coreworld.WorldMinY+1 || y > coreworld.WorldMaxY-1 {
			continue
		}
		support, loaded := world.BlockIfLoaded(x, y-1, z)
		if !loaded {
			return 0, false
		}
		if !coreworld.IsEntitySupportBlock(support.ResourceLocation()) {
			continue
		}
		// Walk nodes are block-centred: all four corners of the 0.6-wide
		// collision box occupy this same column. Read feet and head once.
		feet, feetLoaded := world.BlockIfLoaded(x, y, z)
		head, headLoaded := world.BlockIfLoaded(x, y+1, z)
		if feetLoaded && headLoaded && !coreworld.IsEntitySupportBlock(feet.ResourceLocation()) && !coreworld.IsEntitySupportBlock(head.ResourceLocation()) {
			return y, true
		}
	}
	return 0, false
}

func reconstruct(end *searchNode) []spatial.Vec3 {
	reversed := make([]spatial.Vec3, 0, 16)
	for node := end; node != nil && node.parent != nil; node = node.parent {
		reversed = append(reversed, spatial.Vec3{X: float64(node.key.x) + 0.5, Y: float64(node.key.y), Z: float64(node.key.z) + 0.5})
	}
	path := make([]spatial.Vec3, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return simplify(path)
}

func simplify(path []spatial.Vec3) []spatial.Vec3 {
	if len(path) < 3 {
		return path
	}
	out := make([]spatial.Vec3, 0, len(path))
	out = append(out, path[0])
	for i := 1; i < len(path)-1; i++ {
		previous, current, next := out[len(out)-1], path[i], path[i+1]
		dx1, dy1, dz1 := sign(current.X-previous.X), sign(current.Y-previous.Y), sign(current.Z-previous.Z)
		dx2, dy2, dz2 := sign(next.X-current.X), sign(next.Y-current.Y), sign(next.Z-current.Z)
		if dx1 == dx2 && dy1 == dy2 && dz1 == dz2 {
			continue
		}
		out = append(out, current)
	}
	return append(out, path[len(path)-1])
}

func heuristic(a, b nodeKey) float64 {
	dx, dy, dz := float64(a.x-b.x), float64(a.y-b.y), float64(a.z-b.z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func movementCost(a, b nodeKey) float64 {
	dx, dy, dz := abs(a.x-b.x), abs(a.y-b.y), abs(a.z-b.z)
	cost := 1.0
	if dx != 0 && dz != 0 {
		cost = math.Sqrt2
	}
	return cost + float64(dy)*0.5
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func sign(value float64) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}
