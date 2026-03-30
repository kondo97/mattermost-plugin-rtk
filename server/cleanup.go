package main

// runCleanupLoop is a placeholder for periodic stale-participant cleanup.
// TODO: implement reconciliation with RTK meeting state.
func (p *Plugin) runCleanupLoop(stop chan struct{}) {
	<-stop
}
