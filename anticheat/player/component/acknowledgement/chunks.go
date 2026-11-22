package acknowledgement

import (
	"bytes"
	"fmt"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/oomph-ac/oomph/anticheat/game"
	"github.com/oomph-ac/oomph/anticheat/internal"
	"github.com/oomph-ac/oomph/anticheat/player"
	oworld "github.com/oomph-ac/oomph/anticheat/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ChunkUpdate is an acknowledgment that runs when a player receives a LevelChunk packet from the server.
type ChunkUpdate struct {
	mPlayer *player.Player
	pk      *packet.LevelChunk
}

func NewChunkUpdateACK(p *player.Player, pk *packet.LevelChunk) *ChunkUpdate {
	return &ChunkUpdate{mPlayer: p, pk: pk}
}

func (ack *ChunkUpdate) Run() {
	if ack.pk.CacheEnabled {
		ack.mPlayer.Disconnect(game.ErrorChunkCacheUnsupported)
		return
	}
	cInfo, err := oworld.CacheChunk(ack.pk, ack.mPlayer.BlockNetwork())
	if err != nil {
		ack.mPlayer.Disconnect(fmt.Sprintf(game.ErrorInternalDecodeChunk, err))
		return
	}
	ack.mPlayer.World().AddChunk(ack.pk.Position, cInfo)
	ack.mPlayer.Dbg.Notify(player.DebugModeChunks, true, "added chunk at %v", ack.pk.Position)
}

// SubChunkUpdate is an acknowledgment that runs when a player receives a SubChunk packet.
type SubChunkUpdate struct {
	mPlayer *player.Player
	pk      *packet.SubChunk
}

func NewSubChunkUpdateACK(p *player.Player, pk *packet.SubChunk) *SubChunkUpdate {
	return &SubChunkUpdate{mPlayer: p, pk: pk}
}

func (ack *SubChunkUpdate) Run() {
	if ack.pk.CacheEnabled {
		ack.mPlayer.Disconnect(game.ErrorChunkCacheUnsupported)
		return
	}

	buf := internal.BufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		internal.BufferPool.Put(buf)
	}()
	var bufUsed bool

	terrain := ack.mPlayer.World()
	for _, entry := range ack.pk.SubChunkEntries {
		chunkPos := protocol.ChunkPos{
			ack.pk.Position[0] + int32(entry.Offset[0]),
			ack.pk.Position[2] + int32(entry.Offset[2]),
		}
		ch := terrain.Chunk(chunkPos)
		if ch == nil {
			dimension, ok := world.DimensionByID(int(ack.pk.Dimension))
			if !ok {
				dimension = world.Overworld
			}
			ch = chunk.New(oworld.BlockRegistry, dimension.Range())
			terrain.AddPartialChunk(chunkPos, ch)
			ack.mPlayer.Dbg.Notify(player.DebugModeChunks, true, "new chunk at %v", chunkPos)
		}

		switch entry.Result {
		case protocol.SubChunkResultSuccess:
			payload, ok := entry.RawPayload.Value()
			if !ok {
				ack.mPlayer.Disconnect(fmt.Sprintf(game.ErrorInternalDecodeChunk, "missing subchunk payload"))
				continue
			}
			if bufUsed {
				buf.Reset()
			}
			bufUsed = true
			buf.Write(payload)

			cachedSub, err := oworld.CacheSubChunk(buf, ch, chunkPos, ack.mPlayer.BlockNetwork())
			if err != nil {
				ack.mPlayer.Disconnect(fmt.Sprintf(game.ErrorInternalDecodeChunk, err))
				continue
			}
			ch.Sub()[cachedSub.Layer()] = cachedSub.SubChunk()
			terrain.AddSubChunk(chunkPos, cachedSub.Hash())
			terrain.MarkSubChunkKnown(chunkPos, cachedSub.Layer())
			ack.mPlayer.Dbg.Notify(player.DebugModeChunks, true, "cached subchunk %d at %v", cachedSub.Layer(), chunkPos)
		case protocol.SubChunkResultSuccessAllAir:
			y := (int(ack.pk.Position[1]) + int(entry.Offset[1])) * 16
			if r := ch.Range(); y >= r[0] && y <= r[1] {
				terrain.MarkSubChunkKnown(chunkPos, byte(ch.SubIndex(int16(y))))
			}
			ack.mPlayer.Dbg.Notify(player.DebugModeChunks, true, "all-air chunk at %v", chunkPos)
		default:
			ack.mPlayer.Dbg.Notify(player.DebugModeChunks, true, "no subchunk data for %v (result=%d)", chunkPos, entry.Result)
			continue
		}
	}
}
