// Package bedrock will contain the Minecraft Bedrock Edition protocol adapter
// for GoCraft. It is a planned future milestone and currently a placeholder.
//
// # Planned implementation references
//
//   - Nukkit  — Java Bedrock server (architectural reference)
//     https://github.com/CloudburstMC/Nukkit
//
//   - Dragonfly — Go Bedrock server (Go architecture reference)
//     https://github.com/df-mc/dragonfly
//
//   - gophertunnel — Go Bedrock protocol library (networking/packets)
//     https://github.com/Sandertv/gophertunnel
//
// # Architecture (to be implemented in Milestone 14)
//
// Bedrock clients connect over UDP using the RakNet protocol.
// The adapter will:
//  1. Accept UDP datagrams on a configurable port (default 19132).
//  2. Handle RakNet handshake (OpenConnectionRequest 1 & 2, etc.).
//  3. Perform Bedrock login + Xbox Live / XUID authentication.
//  4. Translate Bedrock packets into canonical core/player and core/world
//     mutations — the same mutations the Java adapter makes.
//  5. Convert core state changes back to Bedrock-specific packets.
//
// The game core (core/) must not import this package.
package bedrock
