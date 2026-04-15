package streamer

// EventHandler defines callbacks that users can implement for streamer events.
type EventHandler interface {
	OnDynamicObjectMoved(objectID int32)
	OnPlayerEditDynamicObject(playerID, objectID int32, response int32, x, y, z, rx, ry, rz float32)
	OnPlayerSelectDynamicObject(playerID, objectID int32, modelID int32, x, y, z float32)
	OnPlayerShootDynamicObject(playerID, objectID int32, weaponID int32, x, y, z float32)
	OnPlayerPickUpDynamicPickup(playerID, pickupID int32)
	OnPlayerEnterDynamicCheckpoint(playerID, checkpointID int32)
	OnPlayerLeaveDynamicCheckpoint(playerID, checkpointID int32)
	OnPlayerEnterDynamicRaceCP(playerID, raceCPID int32)
	OnPlayerLeaveDynamicRaceCP(playerID, raceCPID int32)
	OnPlayerEnterDynamicArea(playerID, areaID int32)
	OnPlayerLeaveDynamicArea(playerID, areaID int32)
	OnDynamicObjectStreamIn(objectID, playerID int32)
	OnDynamicObjectStreamOut(objectID, playerID int32)
	OnDynamicActorStreamIn(actorID, playerID int32)
	OnDynamicActorStreamOut(actorID, playerID int32)
	OnPlayerGiveDamageDynamicActor(playerID, actorID int32, amount float32, weaponID, bodyPart int32)
}

// BaseEventHandler provides no-op defaults for all streamer events.
// Embed this in your handler to only implement the events you need.
type BaseEventHandler struct{}

func (BaseEventHandler) OnDynamicObjectMoved(int32) {}
func (BaseEventHandler) OnPlayerEditDynamicObject(int32, int32, int32, float32, float32, float32, float32, float32, float32) {
}
func (BaseEventHandler) OnPlayerSelectDynamicObject(int32, int32, int32, float32, float32, float32) {}
func (BaseEventHandler) OnPlayerShootDynamicObject(int32, int32, int32, float32, float32, float32)  {}
func (BaseEventHandler) OnPlayerPickUpDynamicPickup(int32, int32)                                   {}
func (BaseEventHandler) OnPlayerEnterDynamicCheckpoint(int32, int32)                                {}
func (BaseEventHandler) OnPlayerLeaveDynamicCheckpoint(int32, int32)                                {}
func (BaseEventHandler) OnPlayerEnterDynamicRaceCP(int32, int32)                                    {}
func (BaseEventHandler) OnPlayerLeaveDynamicRaceCP(int32, int32)                                    {}
func (BaseEventHandler) OnPlayerEnterDynamicArea(int32, int32)                                      {}
func (BaseEventHandler) OnPlayerLeaveDynamicArea(int32, int32)                                      {}
func (BaseEventHandler) OnDynamicObjectStreamIn(int32, int32)                                       {}
func (BaseEventHandler) OnDynamicObjectStreamOut(int32, int32)                                      {}
func (BaseEventHandler) OnDynamicActorStreamIn(int32, int32)                                        {}
func (BaseEventHandler) OnDynamicActorStreamOut(int32, int32)                                       {}
func (BaseEventHandler) OnPlayerGiveDamageDynamicActor(int32, int32, float32, int32, int32)         {}

// Compile-time check.
var _ EventHandler = BaseEventHandler{}
