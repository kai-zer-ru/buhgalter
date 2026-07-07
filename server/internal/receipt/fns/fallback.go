package fns

// buildQRFallback returns a minimal receipt from QR fields when FNS item API is unreachable.
// kkt-online validation (when available) confirms the fiscal key is genuine.
func buildQRFallback(p QRParams) ParsedReceipt {
	total := p.TotalSumKopecks
	return ParsedReceipt{
		FiscalDriveNumber:    p.FiscalDriveNumber,
		FiscalDocumentNumber: p.FiscalDocumentNumber,
		FiscalSign:           p.FiscalSign,
		DateTime:             p.DateTime,
		TotalSum:             total,
		StoreName:            "",
		Items: []Item{{
			Name:     "Покупка по чеку",
			Quantity: 1,
			Price:    total,
			Sum:      total,
		}},
		FNSCheckURL: CheckURL(p),
	}
}
