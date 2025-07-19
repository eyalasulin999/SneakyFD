package beacon

type BeaconType byte

const (
	HELLO_BEACON                BeaconType = 0x01
	HANDLE_PROCESS_BEACON       BeaconType = 0x02
	KILL_PROCESS_FAILED_BEACON  BeaconType = 0x03
	KILLED_PROCESS_BEACON       BeaconType = 0x04
	WAIT_PROCESS_BEACON         BeaconType = 0x05
	WAIT_PROCESS_DONE_BEACON    BeaconType = 0x06
	WAIT_PROCESS_TIMEOUT_BEACON BeaconType = 0x07
	READY_BEACON                BeaconType = 0xFF
)
