package datadog

import (
	"context"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/trace"
)

type Tracer struct {
}

func SetupTrace() {
	ctx := context.Background()
	trace.SetupTrace(ctx, config.OTELConfig{
		ServiceName: "",
	})
}

//  DD_INTERNAL_POD_UID:                           (v1:metadata.uid)
//       DD_EXTERNAL_ENV:                              it-false,cn-service-payment-migration,pu-$(DD_INTERNAL_POD_UID)
//       DD_ENTITY_ID:                                  (v1:metadata.uid)
//       DD_DOGSTATSD_URL:                             unix:///var/run/datadog/dsd.socket
//       DD_TRACE_AGENT_URL:                           unix:///var/run/datadog/apm.sockets
