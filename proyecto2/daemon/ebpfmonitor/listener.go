package ebpfmonitor

import (
	"errors"

	"github.com/cilium/ebpf/ringbuf"
)

/*
Listen mantiene la lectura del Ring Buffer
durante toda la vida del Daemon.
*/
func (monitor *Monitor) Listen(
	handler func(Event),
) error {
	for {
		event, err :=
			monitor.ReadEvent()

		if err != nil {

			if errors.Is(
				err,
				ringbuf.ErrClosed,
			) {
				return nil
			}

			return err
		}

		handler(event)
	}
}
