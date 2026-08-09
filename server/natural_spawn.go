package server

import (
	"math"
	"math/rand"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

// The category layout and limits are generated from the same Pumpkin data
// model used by natural_spawner.rs. The 17x17 (289 chunk) denominator is the
// cap-scaling constant used by Pumpkin and vanilla.
const pumpkinSpawnableChunkDenominator = 17 * 17

// Pumpkin performs a separate creature pass while chunks are generated. Work
// through only a couple of newly active chunks per tick so loading a player's
// complete view cannot cause a single large tick spike.
const (
	pumpkinCreaturePopulationChunksPerTick = 2
	pumpkinMaxCreatureGroupsPerChunk       = 8
	pumpkinCreaturePopulationRadius        = 96.0
	pumpkinCreaturePopulationPerArea       = 16
)

type mobCategory uint8

const (
	mobCategoryMonster mobCategory = iota
	mobCategoryCreature
	mobCategoryAmbient
	mobCategoryAxolotls
	mobCategoryUndergroundWaterCreature
	mobCategoryWaterCreature
	mobCategoryWaterAmbient
	mobCategoryMisc
)

type mobCategorySettings struct {
	max             int
	friendly        bool
	persistentGroup bool
	despawnDistance float64
}

var pumpkinMobCategories = [...]mobCategorySettings{
	{max: 70, despawnDistance: 128},
	{max: 10, friendly: true, persistentGroup: true, despawnDistance: 128},
	{max: 15, friendly: true, despawnDistance: 128},
	{max: 5, friendly: true, despawnDistance: 128},
	{max: 5, friendly: true, despawnDistance: 128},
	{max: 5, friendly: true, persistentGroup: true, despawnDistance: 128},
	{max: 20, friendly: true, despawnDistance: 64},
	{max: -1, friendly: true, persistentGroup: true, despawnDistance: 128},
}

type spawnLocation uint8

const (
	spawnLocationUnrestricted spawnLocation = iota
	spawnLocationInWater
	spawnLocationOnGround
	spawnLocationInLava
)

type pumpkinSpawnEntry struct {
	entityType         string
	minCount, maxCount int
}

type pumpkinBiomeSpawnSettings struct {
	creatureProbability float64
	groups              map[mobCategory][]pumpkinSpawnEntry
}

type pumpkinEntitySpawnSettings struct {
	category              mobCategory
	location              spawnLocation
	limitPerChunk         int
	summonable            bool
	canSpawnFarFromPlayer bool
	movementSpeed         float64
	followRange           float64
	attackDamage          float64
}

type naturalSpawnPlayer struct {
	id       int32
	position spatial.Vec3
}

type naturalSpawnState struct {
	spawnableChunkCount int
	counts              [8]int
	localCounts         map[int32][8]int
	players             []naturalSpawnPlayer
}

func entityWithinSimulationRange(entity *corentity.Entity, players []naturalSpawnPlayer, distance float64) bool {
	if entity == nil {
		return false
	}
	limitSquared := distance * distance
	for _, candidate := range players {
		dx := entity.Position.X - candidate.position.X
		dy := entity.Position.Y - candidate.position.Y
		dz := entity.Position.Z - candidate.position.Z
		if dx*dx+dy*dy+dz*dz <= limitSquared {
			return true
		}
	}
	return false
}

// despawnDistantNaturalMobs provides the chunk-unload lifecycle that GoCraft's
// global entity manager otherwise lacks. Generated villagers, summoned mobs,
// boats, and player-created entities are deliberately excluded.
func (s *Server) despawnDistantNaturalMobs(players []naturalSpawnPlayer, removedIDs *[]int32) {
	for _, entity := range s.world.Entities.Snapshot() {
		if entity.Dead || !entity.NaturalSpawned || entity.Tamed || entity.HasTameOwner {
			continue
		}
		settings, ok := pumpkinEntitySpawnSettingsByType[string(entity.Type)]
		if !ok || settings.category == mobCategoryMisc {
			continue
		}
		distance := pumpkinMobCategories[settings.category].despawnDistance
		if len(players) != 0 && entityWithinSimulationRange(entity, players, distance) {
			continue
		}
		s.world.Entities.Remove(entity.EntityID)
		delete(s.mobAIs, entity.EntityID)
		*removedIDs = append(*removedIDs, entity.EntityID)
	}
}

func (state *naturalSpawnState) globalCap(category mobCategory) int {
	settings := pumpkinMobCategories[category]
	return settings.max * state.spawnableChunkCount / pumpkinSpawnableChunkDenominator
}

func (state *naturalSpawnState) canSpawnGlobal(category mobCategory) bool {
	return state.counts[category] < state.globalCap(category)
}

func (state *naturalSpawnState) playersNearChunk(cx, cz int32) []int32 {
	centerX, centerZ := float64(cx*16+8), float64(cz*16+8)
	near := make([]int32, 0, len(state.players))
	for _, candidate := range state.players {
		dx, dz := centerX-candidate.position.X, centerZ-candidate.position.Z
		if dx*dx+dz*dz < 128*128 {
			near = append(near, candidate.id)
		}
	}
	return near
}

func (state *naturalSpawnState) canSpawnLocal(category mobCategory, cx, cz int32) bool {
	for _, playerID := range state.playersNearChunk(cx, cz) {
		if state.localCounts[playerID][category] < pumpkinMobCategories[category].max {
			return true
		}
	}
	return false
}

func (state *naturalSpawnState) add(category mobCategory, cx, cz int32) {
	state.counts[category]++
	for _, playerID := range state.playersNearChunk(cx, cz) {
		counts := state.localCounts[playerID]
		counts[category]++
		state.localCounts[playerID] = counts
	}
}

// tickNaturalSpawning follows Pumpkin's natural-spawner flow: build the set of
// loaded simulation chunks, scale category caps by that set, enable persistent
// friendly categories every 400 ticks, then make a spawn attempt per eligible
// category and chunk using Pumpkin's generated biome pack tables.
func (s *Server) tickNaturalSpawning() {
	players := s.naturalSpawnPlayers()
	if len(players) == 0 {
		return
	}
	chunks := s.spawnableChunks(players)
	if len(chunks) == 0 {
		return
	}

	s.populatePumpkinGenerationCreatures(chunks, players)
	state := s.newNaturalSpawnState(players, chunks)
	categories := filteredSpawningCategories(&state, s.cfg == nil || s.cfg.Difficulty != "peaceful", s.worldAge%400 == 0)
	if len(categories) == 0 {
		return
	}
	s.spawnRNG.Shuffle(len(chunks), func(i, j int) { chunks[i], chunks[j] = chunks[j], chunks[i] })
	isNight := pumpkinNaturalSpawnNight(s.worldAge)
	for _, chunk := range chunks {
		for _, category := range categories {
			// Preserve half of the hostile cap for the night surface. Without a
			// block-light engine, cave attempts fill the complete cap during the
			// day before any player can see a surface spawn after /time night.
			if category == mobCategoryMonster && !isNight && state.counts[category] >= state.globalCap(category)/2 {
				continue
			}
			if !state.canSpawnGlobal(category) {
				continue
			}
			if state.canSpawnLocal(category, chunk[0], chunk[1]) {
				s.spawnCategoryForChunk(&state, category, chunk[0], chunk[1])
			}
		}
	}
}

// populatePumpkinGenerationCreatures mirrors Pumpkin's
// spawn_mobs_for_chunk_generation pass. Runtime natural spawning is capped at
// ten CREATURE mobs for a full 17x17 simulation area, which is too small to
// represent every four-animal herd. The generation pass is what supplies the
// normal mix of sheep, pigs, chickens, cows, and biome-specific animals.
func (s *Server) populatePumpkinGenerationCreatures(chunks [][2]int32, players []naturalSpawnPlayer) {
	if s.world == nil || s.game == nil {
		return
	}
	if s.creaturePopulatedChunks == nil {
		s.creaturePopulatedChunks = make(map[[2]int32]struct{})
	}
	// GoCraft does not persist entities with chunks yet. Forget population
	// markers only after their chunk has left despawn range; revisiting then
	// deterministically recreates the same bounded simulation population.
	for chunk := range s.creaturePopulatedChunks {
		if !pumpkinChunkWithinPlayerDistance(chunk[0], chunk[1], players, pumpkinMobCategories[mobCategoryCreature].despawnDistance) {
			delete(s.creaturePopulatedChunks, chunk)
		}
	}
	populationCount, typeCounts := s.activeCreaturePopulation(players)
	simulationAreas := (len(chunks) + pumpkinSpawnableChunkDenominator - 1) / pumpkinSpawnableChunkDenominator
	if simulationAreas < 1 {
		simulationAreas = 1
	}
	populationLimit := pumpkinCreaturePopulationPerArea * simulationAreas
	if populationCount >= populationLimit {
		return
	}
	processed := 0
	for _, chunk := range chunks {
		if processed >= pumpkinCreaturePopulationChunksPerTick {
			return
		}
		key := [2]int32{chunk[0], chunk[1]}
		if _, populated := s.creaturePopulatedChunks[key]; populated {
			continue
		}
		if !pumpkinChunkWithinPlayerDistance(chunk[0], chunk[1], players, pumpkinCreaturePopulationRadius) {
			continue
		}
		s.creaturePopulatedChunks[key] = struct{}{}
		processed++
		spawned := s.spawnPumpkinGenerationCreaturesForChunk(
			chunk[0], chunk[1], populationLimit-populationCount, typeCounts,
		)
		populationCount += spawned
		if populationCount >= populationLimit {
			return
		}
	}
}

func (s *Server) activeCreaturePopulation(players []naturalSpawnPlayer) (int, map[string]int) {
	counts := make(map[string]int)
	total := 0
	for _, entity := range s.world.Entities.Snapshot() {
		if entity.Dead || !entityWithinSimulationRange(entity, players, pumpkinMobCategories[mobCategoryCreature].despawnDistance) {
			continue
		}
		settings, ok := pumpkinEntitySpawnSettingsByType[string(entity.Type)]
		if !ok || settings.category != mobCategoryCreature {
			continue
		}
		total++
		counts[string(entity.Type)]++
	}
	return total, counts
}

func pumpkinChunkWithinPlayerDistance(cx, cz int32, players []naturalSpawnPlayer, distance float64) bool {
	centerX, centerZ := float64(cx*16+8), float64(cz*16+8)
	limitSquared := distance * distance
	for _, candidate := range players {
		dx, dz := centerX-candidate.position.X, centerZ-candidate.position.Z
		if dx*dx+dz*dz <= limitSquared {
			return true
		}
	}
	return false
}

func (s *Server) spawnPumpkinGenerationCreaturesForChunk(cx, cz int32, remaining int, typeCounts map[string]int) int {
	if remaining <= 0 {
		return 0
	}
	centerX, centerZ := int(cx)*16+8, int(cz)*16+8
	surfaceY, loaded := s.world.SurfaceYIfLoaded(centerX, centerZ)
	if !loaded {
		return 0
	}
	biome := s.world.BiomeAt(centerX, pumpkinSurfaceSpawnY(s.world, centerX, centerZ, surfaceY), centerZ)
	settings, ok := pumpkinBiomeSpawns[biome]
	if !ok || settings.creatureProbability <= 0 {
		return 0
	}
	entries := settings.groups[mobCategoryCreature]
	if len(entries) == 0 {
		return 0
	}

	worldSeed := int64(0)
	if s.cfg != nil {
		worldSeed = s.cfg.WorldSeed
	}
	rng := rand.New(rand.NewSource(pumpkinCreaturePopulationSeed(worldSeed, cx, cz)))
	spawned := 0
	for group := 0; group < pumpkinMaxCreatureGroupsPerChunk; group++ {
		if rng.Float64() >= settings.creatureProbability {
			break
		}
		entry, supported := pumpkinLeastRepresentedCreatureEntry(entries, typeCounts)
		if !supported {
			continue
		}
		entitySettings, known := pumpkinEntitySpawnSettingsByType[entry.entityType]
		if !known || !entitySettings.summonable || entitySettings.category != mobCategoryCreature {
			continue
		}
		count := entry.minCount
		if entry.maxCount > entry.minCount {
			count += rng.Intn(entry.maxCount - entry.minCount + 1)
		}
		x, z := int(cx)*16+rng.Intn(16), int(cz)*16+rng.Intn(16)
		startX, startZ := x, z
		for range count {
			if spawned >= remaining {
				return spawned
			}
			for attempt := 0; attempt < 4; attempt++ {
				success := false
				candidateSurface, candidateLoaded := s.world.SurfaceYIfLoaded(x, z)
				if candidateLoaded {
					y := pumpkinSurfaceSpawnY(s.world, x, z, candidateSurface)
					if s.validPumpkinGenerationCreaturePosition(entry.entityType, entitySettings, cx, cz, x, y, z) {
						e := corentity.New(s.game.NextEntityID(), newRandomUUID(), corentity.EntityType(entry.entityType),
							float64(x)+0.5, float64(y), float64(z)+0.5)
						// The entity manager is global rather than chunk-persistent, so use
						// the natural distance-based unload lifecycle. The deterministic
						// chunk pass recreates them when that area becomes active again.
						e.NaturalSpawned = true
						e.OnGround = true
						e.Yaw = rng.Float32() * 360
						s.world.Entities.Add(e)
						if s.sessions != nil {
							handler.BroadcastSpawnMob(e, s.sessions)
						}
						spawned++
						typeCounts[entry.entityType]++
						success = true
					}
				}
				x += rng.Intn(5) - rng.Intn(5)
				z += rng.Intn(5) - rng.Intn(5)
				positionCX, positionCZ := coreworld.ChunkCoordsFor(x, z)
				if positionCX != cx || positionCZ != cz {
					x, z = startX, startZ
				}
				if success {
					break
				}
			}
		}
	}
	return spawned
}

func pumpkinLeastRepresentedCreatureEntry(entries []pumpkinSpawnEntry, typeCounts map[string]int) (pumpkinSpawnEntry, bool) {
	var selected pumpkinSpawnEntry
	selectedCount := int(^uint(0) >> 1)
	found := false
	for _, entry := range entries {
		if !pumpkinSpawnEntitySupportedByProtocol(entry.entityType) {
			continue
		}
		count := typeCounts[entry.entityType]
		if !found || count < selectedCount {
			selected, selectedCount, found = entry, count, true
		}
	}
	return selected, found
}

func (s *Server) validPumpkinGenerationCreaturePosition(entityType string, settings pumpkinEntitySpawnSettings, cx, cz int32, x, y, z int) bool {
	positionCX, positionCZ := coreworld.ChunkCoordsFor(x, z)
	if positionCX != cx || positionCZ != cz || settings.location != spawnLocationOnGround {
		return false
	}
	if !pumpkinSpawnLocationOK(s.world, x, y, z, settings.location) ||
		!s.pumpkinEntitySpawnRuleOK(corentity.EntityType(entityType), x, y, z) {
		return false
	}
	ok, loaded := s.world.CanEntityOccupyIfLoaded(float64(x)+0.5, float64(y), float64(z)+0.5)
	return loaded && ok
}

func pumpkinCreaturePopulationSeed(worldSeed int64, cx, cz int32) int64 {
	return worldSeed ^ int64(cx)*341873128712 ^ int64(cz)*132897987541 ^ 0x4352454154555245
}

func (s *Server) naturalSpawnPlayers() []naturalSpawnPlayer {
	players := make([]naturalSpawnPlayer, 0, s.game.OnlineCount())
	s.game.OnlinePlayers(func(p *player.Player) {
		if p.GameMode == player.GameModeSpectator || p.Dead {
			return
		}
		players = append(players, naturalSpawnPlayer{id: p.EntityID, position: p.Position})
	})
	return players
}

func (s *Server) spawnableChunks(players []naturalSpawnPlayer) [][2]int32 {
	distance := 8
	if s.cfg != nil {
		distance = s.cfg.ViewDistance
	}
	unique := make(map[[2]int32]struct{})
	for _, p := range players {
		centerX, centerZ := coreworld.ChunkCoordsFor(int(math.Floor(p.position.X)), int(math.Floor(p.position.Z)))
		for dx := -distance; dx <= distance; dx++ {
			for dz := -distance; dz <= distance; dz++ {
				cx, cz := centerX+int32(dx), centerZ+int32(dz)
				if s.world.IsChunkLoaded(cx, cz) {
					unique[[2]int32{cx, cz}] = struct{}{}
				}
			}
		}
	}
	chunks := make([][2]int32, 0, len(unique))
	for chunk := range unique {
		chunks = append(chunks, chunk)
	}
	return chunks
}

func (s *Server) newNaturalSpawnState(players []naturalSpawnPlayer, chunks [][2]int32) naturalSpawnState {
	state := naturalSpawnState{
		spawnableChunkCount: len(chunks),
		localCounts:         make(map[int32][8]int, len(players)),
		players:             players,
	}
	active := make(map[[2]int32]struct{}, len(chunks))
	for _, chunk := range chunks {
		active[chunk] = struct{}{}
	}
	for _, e := range s.world.Entities.Snapshot() {
		if e.Dead {
			continue
		}
		settings, ok := pumpkinEntitySpawnSettingsByType[string(e.Type)]
		if !ok || settings.category == mobCategoryMisc {
			continue
		}
		cx, cz := coreworld.ChunkCoordsFor(int(math.Floor(e.Position.X)), int(math.Floor(e.Position.Z)))
		if _, ok := active[[2]int32{cx, cz}]; !ok {
			continue
		}
		state.add(settings.category, cx, cz)
	}
	return state
}

func filteredSpawningCategories(state *naturalSpawnState, spawnEnemies, spawnPassives bool) []mobCategory {
	categories := make([]mobCategory, 0, len(pumpkinMobCategories))
	for category := mobCategoryMonster; category <= mobCategoryMisc; category++ {
		settings := pumpkinMobCategories[category]
		if !settings.friendly && !spawnEnemies {
			continue
		}
		if settings.persistentGroup && !spawnPassives {
			continue
		}
		if state.canSpawnGlobal(category) {
			categories = append(categories, category)
		}
	}
	return categories
}

func (s *Server) spawnCategoryForChunk(state *naturalSpawnState, category mobCategory, cx, cz int32) {
	x := int(cx)*16 + s.spawnRNG.Intn(16)
	z := int(cz)*16 + s.spawnRNG.Intn(16)
	surfaceY, loaded := s.world.SurfaceYIfLoaded(x, z)
	if !loaded || surfaceY <= coreworld.WorldMinY {
		return
	}
	y := coreworld.WorldMinY + s.spawnRNG.Intn(surfaceY-coreworld.WorldMinY+2)
	surfaceAttempt := category == mobCategoryCreature || category == mobCategoryMonster && pumpkinNaturalSpawnNight(s.worldAge)
	if surfaceAttempt {
		y = pumpkinSurfaceSpawnY(s.world, x, z, surfaceY)
	}
	if y <= coreworld.WorldMinY {
		return
	}

	clusterSize := 0
	for groupAttempt := 0; groupAttempt < 3; groupAttempt++ {
		newX, newZ := x, z
		biome := s.world.BiomeAt(x, y, z)
		entries := pumpkinBiomeSpawns[biome].groups[category]
		if len(entries) == 0 {
			return
		}
		entry, ok := s.pumpkinSpawnEntrySupportedByProtocol(entries)
		if !ok {
			continue
		}
		groupSize := entry.minCount
		if entry.maxCount > entry.minCount {
			groupSize += s.spawnRNG.Intn(entry.maxCount - entry.minCount + 1)
		}
		entitySettings, ok := pumpkinEntitySpawnSettingsByType[entry.entityType]
		if !ok || !entitySettings.summonable {
			continue
		}

		for attempt := 0; attempt < groupSize; attempt++ {
			newX += s.spawnRNG.Intn(6) - s.spawnRNG.Intn(6)
			newZ += s.spawnRNG.Intn(6) - s.spawnRNG.Intn(6)
			newY := adjustPumpkinSpawnY(s.world, newX, y, newZ, entitySettings.location)
			if surfaceAttempt && entitySettings.location == spawnLocationOnGround {
				if candidateSurface, candidateLoaded := s.world.SurfaceYIfLoaded(newX, newZ); candidateLoaded {
					newY = pumpkinSurfaceSpawnY(s.world, newX, newZ, candidateSurface)
				}
			}
			actualCategory := entitySettings.category
			if !s.validNaturalSpawnPosition(state.players, actualCategory, entry.entityType, entitySettings, cx, cz, newX, newY, newZ) {
				continue
			}
			if !state.canSpawnGlobal(actualCategory) || !state.canSpawnLocal(actualCategory, cx, cz) {
				return
			}

			e := corentity.New(s.game.NextEntityID(), newRandomUUID(), corentity.EntityType(entry.entityType),
				float64(newX)+0.5, float64(newY), float64(newZ)+0.5)
			e.NaturalSpawned = true
			e.Yaw = s.spawnRNG.Float32() * 360
			e.OnGround = entitySettings.location == spawnLocationOnGround
			s.world.Entities.Add(e)
			handler.BroadcastSpawnMob(e, s.sessions)
			state.add(actualCategory, cx, cz)
			clusterSize++
			if clusterSize >= entitySettings.limitPerChunk {
				return
			}
		}
	}
}

func (s *Server) pumpkinSpawnEntrySupportedByProtocol(entries []pumpkinSpawnEntry) (pumpkinSpawnEntry, bool) {
	return pumpkinSpawnEntrySupportedByProtocolAt(entries, s.spawnRNG.Intn(len(entries)))
}

func pumpkinSpawnEntrySupportedByProtocolAt(entries []pumpkinSpawnEntry, start int) (pumpkinSpawnEntry, bool) {
	for offset := range len(entries) {
		entry := entries[(start+offset)%len(entries)]
		if !pumpkinSpawnEntitySupportedByProtocol(entry.entityType) {
			continue
		}
		return entry, true
	}
	return pumpkinSpawnEntry{}, false
}

func pumpkinSpawnEntitySupportedByProtocol(entityType string) bool {
	switch entityType {
	case "minecraft:nautilus", "minecraft:parched", "minecraft:sulfur_cube":
		// These are present in the newer Pumpkin checkout but not in
		// GoCraft's negotiated Java 1.21.4 entity registry.
		return false
	default:
		return true
	}
}

func adjustPumpkinSpawnY(world *coreworld.World, x, y, z int, location spawnLocation) int {
	if location != spawnLocationOnGround {
		return y
	}
	below := world.GetBlock(x, y-1, z)
	if !coreworld.IsEntitySupportBlock(below.ResourceLocation()) && !isSpawnFluid(below) {
		return y - 1
	}
	return y
}

func (s *Server) validNaturalSpawnPosition(players []naturalSpawnPlayer, category mobCategory, entityType string, settings pumpkinEntitySpawnSettings, cx, cz int32, x, y, z int) bool {
	positionChunkX, positionChunkZ := coreworld.ChunkCoordsFor(x, z)
	if positionChunkX != cx || positionChunkZ != cz {
		return false
	}
	center := spatial.Vec3{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}
	nearestDistance := math.MaxFloat64
	for _, p := range players {
		dx, dy, dz := center.X-p.position.X, center.Y-p.position.Y, center.Z-p.position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance < nearestDistance {
			nearestDistance = distance
		}
	}
	if nearestDistance <= 24*24 {
		return false
	}
	dx, dz := center.X-(float64(s.spawnX)+0.5), center.Z-(float64(s.spawnZ)+0.5)
	if dx*dx+dz*dz <= 24*24 {
		return false
	}
	maxDistance := pumpkinMobCategories[category].despawnDistance
	if !settings.canSpawnFarFromPlayer && nearestDistance > maxDistance*maxDistance {
		return false
	}
	if !pumpkinSpawnLocationOK(s.world, x, y, z, settings.location) {
		return false
	}
	if !s.pumpkinEntitySpawnRuleOK(corentity.EntityType(entityType), x, y, z) {
		return false
	}
	ok, loaded := s.world.CanEntityOccupyIfLoaded(center.X, center.Y, center.Z)
	return loaded && ok
}

func pumpkinNaturalSpawnNight(worldAge int64) bool {
	timeOfDay := worldAge % 24000
	return timeOfDay >= 13000 && timeOfDay <= 23000
}

func pumpkinSurfaceSpawnY(world *coreworld.World, x, z, highestY int) int {
	groundY := world.GroundYAtOrBelow(x, z, highestY)
	if groundY >= coreworld.WorldMinY {
		return groundY + 1
	}
	return highestY + 1
}

func pumpkinSpawnLocationOK(world *coreworld.World, x, y, z int, location spawnLocation) bool {
	current := world.GetBlock(x, y, z)
	switch location {
	case spawnLocationInLava:
		return current.ResourceLocation() == "minecraft:lava"
	case spawnLocationInWater:
		above := world.GetBlock(x, y+1, z)
		return current.ResourceLocation() == "minecraft:water" && !coreworld.IsEntitySupportBlock(above.ResourceLocation())
	case spawnLocationOnGround:
		below := world.GetBlock(x, y-1, z)
		above := world.GetBlock(x, y+1, z)
		return coreworld.IsEntitySupportBlock(below.ResourceLocation()) &&
			pumpkinEmptySpawnBlock(current) && pumpkinEmptySpawnBlock(above)
	default:
		return true
	}
}

func pumpkinEmptySpawnBlock(block coreworld.Block) bool {
	if coreworld.IsEntitySupportBlock(block.ResourceLocation()) || isSpawnFluid(block) {
		return false
	}
	switch block.ResourceLocation() {
	case "minecraft:fire", "minecraft:soul_fire", "minecraft:wither_rose", "minecraft:sweet_berry_bush":
		return false
	}
	return true
}

func isSpawnFluid(block coreworld.Block) bool {
	name := block.ResourceLocation()
	return name == "minecraft:water" || name == "minecraft:lava" || block.Properties["waterlogged"] == "true"
}

func (s *Server) pumpkinEntitySpawnRuleOK(entityType corentity.EntityType, x, y, z int) bool {
	world := s.world
	if settings, ok := pumpkinEntitySpawnSettingsByType[string(entityType)]; ok && settings.category == mobCategoryCreature {
		surface, loaded := world.SurfaceYIfLoaded(x, z)
		if !loaded || y != pumpkinSurfaceSpawnY(world, x, z, surface) {
			return false
		}
	}
	switch entityType {
	case corentity.TypeSlime:
		if y >= 40 || s.spawnRNG.Intn(10) != 0 {
			return false
		}
		cx, cz := coreworld.ChunkCoordsFor(x, z)
		worldSeed := int64(0)
		if s.cfg != nil {
			worldSeed = s.cfg.WorldSeed
		}
		return pumpkinSlimeChunk(worldSeed, cx, cz)
	case corentity.TypeBogged, corentity.TypeCaveSpider, corentity.TypeCreeper,
		corentity.TypeEnderman, corentity.TypeRavager, corentity.TypeSkeleton,
		corentity.TypeSpider, corentity.TypeWitch, corentity.TypeWither,
		corentity.TypeWitherSkeleton, corentity.TypeZombie, corentity.TypeZombieHorse,
		corentity.TypeZombieVillager, corentity.TypeCreaker, corentity.TypeEvoker,
		corentity.TypeIllusioner, corentity.TypeVex, corentity.TypeVindicator,
		corentity.TypeWarden:
		surface, loaded := world.SurfaceYIfLoaded(x, z)
		if !loaded {
			return false
		}
		// GoCraft does not yet expose a canonical block-light engine. Match
		// Pumpkin's darkness rule for the two reliable cases: covered caves and
		// the night surface.
		return y+1 < surface || pumpkinNaturalSpawnNight(s.worldAge)
	case corentity.TypeBat:
		surface, loaded := world.SurfaceYIfLoaded(x, z)
		return loaded && y < surface
	}
	return true
}

func pumpkinSlimeChunk(worldSeed int64, chunkX, chunkZ int32) bool {
	x, z := int64(chunkX), int64(chunkZ)
	seed := worldSeed + x*x*4_987_142 + x*5_947_611 + z*z*4_392_871 + z*389_711
	seed ^= 987_234_911
	return javaRandomNextInt(seed, 10) == 0
}

func javaRandomNextInt(seed int64, bound int32) int32 {
	const (
		multiplier = uint64(0x5DEECE66D)
		addend     = uint64(0xB)
		mask       = uint64(1<<48 - 1)
	)
	state := (uint64(seed) ^ multiplier) & mask
	next := func(bits uint) int32 {
		state = (state*multiplier + addend) & mask
		return int32(state >> (48 - bits))
	}
	if bound <= 0 {
		return 0
	}
	if bound&(bound-1) == 0 {
		return int32((int64(bound) * int64(next(31))) >> 31)
	}
	for {
		bits := next(31)
		value := bits % bound
		if bits-value+(bound-1) >= 0 {
			return value
		}
	}
}
