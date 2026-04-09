package rns

import (
	"sync"
	"time"
)

type RequestReceiptCallbacks struct {
	Response func(*RequestReceipt)
	Failed   func(*RequestReceipt)
	Progress func(*RequestReceipt)
}

type RequestReceipt struct {
	link *Link

	packetReceipt *PacketReceipt

	mu sync.Mutex

	requestID   []byte
	requestSize int

	response             any
	responseSize         int
	responseTransferSize int
	responseMetadata     any
	responseConcludedAt  time.Time
	startedAt            time.Time
	sentAt               time.Time
	concludedAt          time.Time
	status               byte
	progress             float64
	timeout              time.Duration
	responseTimeoutOnce  sync.Once
	timedOutOnce         sync.Once
	callbacks            RequestReceiptCallbacks
}

func newRequestReceipt(
	link *Link,
	packetReceipt *PacketReceipt,
	requestID []byte,
	timeout float64,
	requestSize int,
	responseCb func(*RequestReceipt),
	failedCb func(*RequestReceipt),
	progressCb func(*RequestReceipt),
) *RequestReceipt {
	rr := &RequestReceipt{
		link:          link,
		packetReceipt: packetReceipt,
		requestID:     nil,
		requestSize:   requestSize,
		status:        ReceiptSent,
		progress:      0,
		sentAt:        time.Now(),
		timeout:       time.Duration(timeout * float64(time.Second)),
		callbacks: RequestReceiptCallbacks{
			Response: responseCb,
			Failed:   failedCb,
			Progress: progressCb,
		},
	}

	if rr.timeout <= 0 {
		rr.timeout = time.Duration(float64(link.RTT)*link.TrafficTimeoutFactor) + time.Duration(1.125*float64(ResponseMaxGraceTime))
	}

	if len(requestID) > 0 {
		rr.requestID = copyBytes(requestID)
	}

	if packetReceipt != nil {
		if rr.requestID == nil {
			rr.requestID = copyBytes(packetReceipt.TruncatedHash)
		}
		rr.startedAt = time.Now()
		packetReceipt.SetTimeout(timeout)
		packetReceipt.SetDeliveryCallback(func(*PacketReceipt) {
			rr.markDelivered()
		})
		packetReceipt.SetTimeoutCallback(func(*PacketReceipt) {
			rr.requestTimedOut()
		})
		// In the in-process integration transport, the delivery proof can arrive
		// synchronously during Packet.Send(), before we attach callbacks here.
		// Mirror Python semantics by observing already-delivered receipts.
		if packetReceipt.Status == ReceiptDelivered {
			rr.markDelivered()
		} else if packetReceipt.Status == ReceiptFailed {
			rr.requestTimedOut()
		}
	}

	link.addPendingRequest(rr)
	return rr
}

func (rr *RequestReceipt) RequestID() []byte {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return copyBytes(rr.requestID)
}

func (rr *RequestReceipt) Status() byte {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.status
}

func (rr *RequestReceipt) Progress() float64 {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.progress
}

func (rr *RequestReceipt) Response() any {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.response
}

func (rr *RequestReceipt) ResponseSize() int {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.responseSize
}

func (rr *RequestReceipt) ResponseTransferSize() int {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.responseTransferSize
}

func (rr *RequestReceipt) ResponseConcludedAt() float64 {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.responseConcludedAt.IsZero() {
		return 0
	}
	return float64(rr.responseConcludedAt.UnixNano()) / 1e9
}

func (rr *RequestReceipt) SentAt() float64 {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.sentAt.IsZero() {
		return 0
	}
	return float64(rr.sentAt.UnixNano()) / 1e9
}

func (rr *RequestReceipt) RequestSize() int {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.requestSize
}

func (rr *RequestReceipt) Concluded() bool {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.status == ReceiptReady || rr.status == ReceiptFailed
}

func (rr *RequestReceipt) Metadata() any {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.responseMetadata
}

func (rr *RequestReceipt) markDelivered() {
	rr.mu.Lock()
	if rr.status == ReceiptFailed || rr.status == ReceiptReady {
		rr.mu.Unlock()
		return
	}
	if rr.startedAt.IsZero() {
		rr.startedAt = time.Now()
	}
	rr.status = ReceiptDelivered
	rr.mu.Unlock()
}

func (rr *RequestReceipt) startResponseWait() {
	rr.responseTimeoutOnce.Do(func() {
		if rr.timeout <= 0 {
			return
		}
		deadline := time.Now().Add(rr.timeout)
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				if rr.status != ReceiptDelivered {
					return
				}
				if time.Now().After(deadline) {
					rr.requestTimedOut()
					return
				}
				<-ticker.C
			}
		}()
	})
}

func (rr *RequestReceipt) noteResponseAdvertisement(adv *ResourceAdvertisement) {
	rr.mu.Lock()
	if rr.responseSize == 0 {
		rr.responseSize = adv.D
	}
	rr.responseTransferSize += adv.T
	if rr.startedAt.IsZero() {
		rr.startedAt = time.Now()
	}
	rr.mu.Unlock()
}

func (rr *RequestReceipt) responseResourceProgress(res *Resource) {
	if res == nil {
		return
	}
	rr.mu.Lock()
	if rr.status == ReceiptFailed {
		rr.mu.Unlock()
		res.Cancel()
		return
	}
	rr.status = ReceiptReceiving
	rr.progress = res.Progress()
	rr.mu.Unlock()
	rr.ensurePacketReceiptDelivered()
	if rr.callbacks.Progress != nil {
		go rr.safeCallback(rr.callbacks.Progress)
	}
}

func (rr *RequestReceipt) responseReceived(resp any, metadata any, transferSize int) {
	rr.mu.Lock()
	if rr.status == ReceiptFailed {
		rr.mu.Unlock()
		return
	}
	rr.response = resp
	if metadata != nil {
		rr.responseMetadata = metadata
	}
	if rr.responseConcludedAt.IsZero() {
		rr.responseConcludedAt = time.Now()
	}
	rr.concludedAt = rr.responseConcludedAt
	rr.progress = 1.0
	rr.status = ReceiptReady
	if transferSize > 0 {
		rr.responseTransferSize = transferSize
	}
	rr.mu.Unlock()
	rr.ensurePacketReceiptDelivered()
	if rr.callbacks.Progress != nil {
		go rr.safeCallback(rr.callbacks.Progress)
	}
	if rr.callbacks.Response != nil {
		go rr.safeCallback(rr.callbacks.Response)
	}
	rr.link.removePendingRequest(rr)
}

func (rr *RequestReceipt) requestTimedOut() {
	rr.timedOutOnce.Do(func() {
		rr.mu.Lock()
		if rr.status != ReceiptDelivered {
			rr.mu.Unlock()
			return
		}
		rr.status = ReceiptFailed
		rr.concludedAt = time.Now()
		rr.mu.Unlock()
		rr.link.removePendingRequest(rr)
		if rr.callbacks.Failed != nil {
			go rr.safeCallback(rr.callbacks.Failed)
		}
	})
}

func (rr *RequestReceipt) ensurePacketReceiptDelivered() {
	rr.mu.Lock()
	pr := rr.packetReceipt
	rr.mu.Unlock()
	if pr == nil {
		return
	}
	if pr.Status != ReceiptDelivered {
		pr.Status = ReceiptDelivered
		pr.Proved = true
		pr.ConcludedAt = time.Now()
		if pr.Callbacks.Delivery != nil {
			go pr.Callbacks.Delivery(pr)
		}
	}
}

func (rr *RequestReceipt) safeCallback(cb func(*RequestReceipt)) {
	defer func() {
		if rec := recover(); rec != nil {
			Log("request receipt callback panic", LOG_ERROR)
		}
	}()
	cb(rr)
}
