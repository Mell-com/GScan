package Spider

import (
	"GScan/infoscan/dao"
	"GScan/pkg/logger"
	"context"
	"sync"
)

func (s *Spider) runWK(ctx context.Context, wg *sync.WaitGroup, maxnum int) {
	logger.PF(logger.LINFO, "<Spider>[%s]启动%d线程", s.Host, maxnum)
	ctx2, cf := context.WithCancel(ctx)
	for i := 0; i < maxnum; i++ {
		go s.worker(ctx2, cf, wg)
	}
	for {
		select {
		case <-ctx2.Done():
			//保留一个
			go s.worker(ctx, nil, wg)
			logger.PF(logger.LINFO, "<Spider>[%s]全部URL任务已完成，保留一个Worker处理其他Spider的外链结果.", s.Host)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Spider) worker(ctx context.Context, ctxfunc context.CancelFunc, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	defer logger.PF(logger.LINFO, "<Spider>[%s]Worker Exit", s.Host)
	workerChan := s.scheduler.WorkerChan()
	for {
		s.scheduler.WorkerReady(workerChan)
		select {
		case page := <-workerChan:
			bytes, err := s.Reqer.GetUrl(page)
			if err != nil {
				page.ErrorNum += 1
				page.Status = "访问出错"
				page.Error = err.Error()
			} else {
				page.Status = "Success"
			}
			s.datapress(ctx, page, bytes)
			s.scheduler.Complete()
			if !s.scheduler.Working() && s.scheduler.GetrunningNum() == 0 {
				if ctxfunc != nil {
					ctxfunc()
				}
			}
		case <-ctx.Done():
			return
			//if s.scheduler.Working() {
			//	logger.PF(logger.LDEBUG, "<Spider>[%s]还有未完成任务，请等待。", s.Host)
			//	time.Sleep(2 * time.Second)
			//} else {
			//	return
			//}
		}
	}

}

func (s *Spider) datapress(ctx context.Context, page *dao.Page, data []byte) {
	s.Processor(page, data)
	s.DataProcessor.Handler(ctx, page, data)
	page = nil
	data = nil //触发GC
	//runtime.GC()
}
