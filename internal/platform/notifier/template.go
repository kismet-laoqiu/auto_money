package notifier

import "strings"

func (notification Notification) Render() string {
	lines := []string{kindHeading(notification.Kind)}
	if notification.Title != "" {
		lines = append(lines, notification.Title)
	}
	if notification.Summary != "" {
		lines = append(lines, notification.Summary)
	}
	lines = append(lines, notification.Details...)
	return strings.Join(lines, "\n")
}

func kindHeading(kind Kind) string {
	switch kind {
	case KindFill:
		return "成交回报"
	case KindRiskHalt:
		return "风控停止"
	case KindPromotionApproved:
		return "上线审批通过"
	case KindCanaryDegraded:
		return "canary 降级"
	case KindDailySummary:
		return "每日总结"
	default:
		return "平台通知"
	}
}
