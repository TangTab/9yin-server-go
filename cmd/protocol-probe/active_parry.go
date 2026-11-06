package main

import (
	"fmt"
	"log"
	"time"
)

// CLIENT_CUSTOMMSG_ACTIVE_PARRY is the shared C->S ingress used by the
// modern client for active-parry controls.  It is deliberately separate from
// 211: individual routines supply the attack data, while parry is a player
// combat-state request that must later be evaluated when an incoming hit is
// resolved.
const clientCustomActiveParry int32 = 218

// activeParryRequest is intentionally lossless for the two fields known from
// custom_sender: subtype and its accompanying argument.  We retain the
// original TVarList values rather than coercing the second value to a guessed
// enum; the live capture will determine its exact gameplay meaning.
type activeParryRequest struct {
	Subtype  int32
	Argument clientCustomValue
	At       time.Time
}

// handleActiveParryCustom establishes the authoritative ingress boundary.
// It accepts all integral encodings used by modern TVarList writers (the
// faculty UI has already demonstrated that some forms emit double enum
// values).  It does not fabricate a visual action or a fixed parry window:
// those depend on the native hit-result path, which has not yet been proven.
func handleActiveParryCustom(player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomActiveParry {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("active parry before scene player exists")
	}
	if len(custom.Values) < 3 {
		return true, fmt.Errorf("active parry requires subtype and argument")
	}
	subtype, ok := custom.Values[1].exactInt32()
	if !ok {
		return true, fmt.Errorf("active parry subtype must be an integral TVarList value, got %s", custom.Values[1])
	}
	request := activeParryRequest{Subtype: subtype, Argument: custom.Values[2], At: time.Now().UTC()}
	player.setActiveParry(request)
	log.Printf("%s: accepted active parry subtype=%d argument=%s", remote, subtype, request.Argument)
	return true, nil
}
