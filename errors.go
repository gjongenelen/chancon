package chancon

import "errors"

var ErrNoSslConfig = errors.New("no ssl-config is set")

var ErrTimedOutWaitingForReply = errors.New("timed out waiting for reply")
