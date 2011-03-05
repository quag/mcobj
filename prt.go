package main

type PrtGenerator struct {
	enclosedsChan chan *EnclosedChunkJob
	completeChan  chan bool
}

func (o *PrtGenerator) Start(outFilename string, total int, maxProcs int) {
	o.enclosedsChan = make(chan *EnclosedChunkJob, maxProcs*2)
	o.completeChan = make(chan bool)

	for i := 0; i < maxProcs; i++ {
		go func() {
			for {
				var job = <-o.enclosedsChan

				if job.last {
					o.completeChan <- true
				}
			}
		}()
	}
}

func (o *PrtGenerator) GetEnclosedJobsChan() chan *EnclosedChunkJob {
	return o.enclosedsChan
}

func (o *PrtGenerator) GetCompleteChan() chan bool {
	return o.completeChan
}

func (o *PrtGenerator) Close() {
}
