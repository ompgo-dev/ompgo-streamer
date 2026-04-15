package streamer

import (
	"context"

	"github.com/ompgo-dev/ompgo/pkg/omp"
	"github.com/ompgo-dev/ompgo/pkg/omp/actor"
	"github.com/ompgo-dev/ompgo/pkg/omp/pickup"
	"github.com/ompgo-dev/ompgo/pkg/omp/playerobject"
	"github.com/ompgo-dev/ompgo/pkg/omp/players"
	"github.com/ompgo-dev/ompgo/pkg/omp/playertextlabel"
)

func (s *Streamer) onTick(_ context.Context, _ *omp.TickEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalTickCount++
	for _, ps := range s.playerStates {
		tickRate := ps.TickRate
		if tickRate <= 0 {
			tickRate = s.cfg.TickRate
		}
		ps.TickCount++
		if ps.TickCount < tickRate {
			continue
		}
		ps.TickCount = 0

		var x, y, z float32
		players.GetPos(ps.Player, &x, &y, &z)
		ps.Position = Vector3{X: x, Y: y, Z: z}
		ps.WorldID = players.GetVirtualWorld(ps.Player)
		ps.Interior = players.GetInterior(ps.Player)
		s.processPlayerUpdate(ps)
	}
	return nil
}

func (s *Streamer) onPlayerConnect(_ context.Context, event *omp.PlayerConnectEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerID := players.GetID(event.Player)
	s.playerStates[playerID] = newPlayerState(playerID, event.Player)
	return nil
}

func (s *Streamer) onPlayerDisconnect(_ context.Context, event *omp.PlayerDisconnectEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}

	for id, serverObj := range ps.Objects {
		playerobject.Destroy(ps.Player, serverObj)
		delete(ps.Objects, id)
	}
	for id, serverTL := range ps.TextLabels {
		playertextlabel.Destroy(ps.Player, serverTL)
		delete(ps.TextLabels, id)
	}
	for id, slot := range ps.MapIcons {
		players.RemoveMapIcon(ps.Player, slot)
		delete(ps.MapIcons, id)
	}

	delete(s.playerStates, playerID)
	return nil
}

func (s *Streamer) onPlayerEnterCheckpoint(_ context.Context, event *omp.PlayerEnterCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerEnterDynamicCheckpoint(playerID, ps.VisibleCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerLeaveCheckpoint(_ context.Context, event *omp.PlayerLeaveCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerLeaveDynamicCheckpoint(playerID, ps.VisibleCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerEnterRaceCheckpoint(_ context.Context, event *omp.PlayerEnterRaceCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleRaceCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerEnterDynamicRaceCP(playerID, ps.VisibleRaceCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerLeaveRaceCheckpoint(_ context.Context, event *omp.PlayerLeaveRaceCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleRaceCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerLeaveDynamicRaceCP(playerID, ps.VisibleRaceCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerPickUpPickup(_ context.Context, event *omp.PlayerPickUpPickupEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	serverPickupID := pickup.GetID(event.Pickup)
	for dynamicID, serverPickup := range s.activePickups {
		if pickup.GetID(serverPickup) == serverPickupID {
			s.cfg.EventHandler.OnPlayerPickUpDynamicPickup(playerID, dynamicID)
			break
		}
	}
	return nil
}

func (s *Streamer) onPlayerGiveDamageActor(_ context.Context, event *omp.PlayerGiveDamageActorEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	serverActorID := actor.GetID(event.Actor)
	for dynamicID, serverActor := range s.activeActors {
		if actor.GetID(serverActor) == serverActorID {
			s.cfg.EventHandler.OnPlayerGiveDamageDynamicActor(playerID, dynamicID, event.Amount, event.Weapon, event.Part)
			break
		}
	}
	return nil
}
