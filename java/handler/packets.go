package handler

// packets.go is the single source of truth for every numeric packet ID used
// by the Java Edition handler layer.
//
// IDs are resolved at startup from the embedded protocol-data JSON files via
// protocoldata.MustCB / MustSB; a misspelled or missing packet name panics
// before the server accepts any connection.
//
// Semantic names follow the vanilla internal naming convention (minecraft:*).
// Updating to a new protocol version means replacing
// internal/protocoldata/java/<version>/*.json and rebuilding — no Go source
// changes are required.

import "GoCraft/internal/protocoldata"

// ── Handshake state ───────────────────────────────────────────────────────────

var (
	packetIDHandshake = protocoldata.MustSB("handshake", "minecraft:intention")
)

// ── Status state ──────────────────────────────────────────────────────────────

var (
	packetIDStatusRequest  = protocoldata.MustSB("status", "minecraft:status_request")
	packetIDStatusResponse = protocoldata.MustCB("status", "minecraft:status_response")
	packetIDPingRequest    = protocoldata.MustSB("status", "minecraft:ping_request")
	packetIDPongResponse   = protocoldata.MustCB("status", "minecraft:pong_response")
)

// ── Login state ───────────────────────────────────────────────────────────────

var (
	// Serverbound
	packetIDLoginStart         = protocoldata.MustSB("login", "minecraft:hello")
	packetIDEncryptionResponse = protocoldata.MustSB("login", "minecraft:key")
	packetIDLoginAcknowledged  = protocoldata.MustSB("login", "minecraft:login_acknowledged")

	// Clientbound
	packetIDLoginDisconnect  = protocoldata.MustCB("login", "minecraft:login_disconnect")
	packetIDEncryptionRequest = protocoldata.MustCB("login", "minecraft:hello")
	packetIDLoginSuccess      = protocoldata.MustCB("login", "minecraft:game_profile")
)

// ── Configuration state ───────────────────────────────────────────────────────

var (
	// Serverbound
	packetIDClientInformation     = protocoldata.MustSB("configuration", "minecraft:client_information")
	packetIDServerboundKnownPacks = protocoldata.MustSB("configuration", "minecraft:select_known_packs")
	packetIDAcknowledgeFinish     = protocoldata.MustSB("configuration", "minecraft:finish_configuration")

	// Clientbound
	packetIDConfigPluginMessage   = protocoldata.MustCB("configuration", "minecraft:custom_payload")
	packetIDFinishConfiguration   = protocoldata.MustCB("configuration", "minecraft:finish_configuration")
	packetIDFeatureFlags          = protocoldata.MustCB("configuration", "minecraft:update_enabled_features")
	packetIDUpdateTags            = protocoldata.MustCB("configuration", "minecraft:update_tags")
	packetIDClientboundKnownPacks = protocoldata.MustCB("configuration", "minecraft:select_known_packs")
)

// ── Play state — clientbound (S→C) ───────────────────────────────────────────

var (
	packetIDPlayLogin        = protocoldata.MustCB("play", "minecraft:login")
	packetIDPlayerAbilities  = protocoldata.MustCB("play", "minecraft:player_abilities")
	packetIDPlayerInfoUpdate = protocoldata.MustCB("play", "minecraft:player_info_update")
	packetIDPlayerInfoRemove = protocoldata.MustCB("play", "minecraft:player_info_remove")
	packetIDSyncPosition     = protocoldata.MustCB("play", "minecraft:player_position")
	packetIDGameEvent        = protocoldata.MustCB("play", "minecraft:game_event")
	packetIDSetCenterChunk   = protocoldata.MustCB("play", "minecraft:set_chunk_cache_center")
	packetIDSpawnPosition    = protocoldata.MustCB("play", "minecraft:set_default_spawn_position")
	packetIDServerKeepAlive  = protocoldata.MustCB("play", "minecraft:keep_alive")
	packetIDForgetLevelChunk = protocoldata.MustCB("play", "minecraft:forget_level_chunk")
	packetIDSystemChatMessage = protocoldata.MustCB("play", "minecraft:system_chat")
	packetIDSpawnEntity      = protocoldata.MustCB("play", "minecraft:spawn_entity")
	packetIDRemoveEntities   = protocoldata.MustCB("play", "minecraft:remove_entities")
	packetIDSetHeadRotation  = protocoldata.MustCB("play", "minecraft:rotate_head")
	packetIDTeleportEntity   = protocoldata.MustCB("play", "minecraft:teleport_entity")
	packetIDBlockUpdate      = protocoldata.MustCB("play", "minecraft:block_update")
	packetIDAcknowledgeBlockChange = protocoldata.MustCB("play", "minecraft:acknowledge_block_change")
	packetIDSetContainerContent    = protocoldata.MustCB("play", "minecraft:set_container_content")
	packetIDSetHeldItemSC    = protocoldata.MustCB("play", "minecraft:set_held_slot")
	packetIDCommands         = protocoldata.MustCB("play", "minecraft:commands")
	packetIDDisconnectPlay   = protocoldata.MustCB("play", "minecraft:disconnect")
)

// ── Play state — serverbound (C→S) ───────────────────────────────────────────

var (
	packetIDConfirmTeleport              = protocoldata.MustSB("play", "minecraft:accept_teleportation")
	packetIDClientKeepAlive             = protocoldata.MustSB("play", "minecraft:keep_alive")
	packetIDSetPlayerPosition            = protocoldata.MustSB("play", "minecraft:move_player_pos")
	packetIDSetPlayerPositionAndRotation = protocoldata.MustSB("play", "minecraft:move_player_pos_rot")
	packetIDSetPlayerRotation            = protocoldata.MustSB("play", "minecraft:move_player_rot")
	packetIDSetPlayerOnGround            = protocoldata.MustSB("play", "minecraft:move_player_status_only")
	packetIDChatCommand                  = protocoldata.MustSB("play", "minecraft:chat_command")
	packetIDChatMessage                  = protocoldata.MustSB("play", "minecraft:chat")
	packetIDPlayerAction                 = protocoldata.MustSB("play", "minecraft:player_action")
	packetIDSetHeldItemCS                = protocoldata.MustSB("play", "minecraft:set_carried_item")
	packetIDCreativeModeSetItem          = protocoldata.MustSB("play", "minecraft:set_creative_mode_slot")
	packetIDUseItemOn                    = protocoldata.MustSB("play", "minecraft:use_item_on")
)
