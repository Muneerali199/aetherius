package model

type NotificationType string

const (
	NotifDeploymentCreated      NotificationType = "deployment.created"
	NotifDeploymentRunning      NotificationType = "deployment.running"
	NotifDeploymentFailed       NotificationType = "deployment.failed"
	NotifDeploymentStopped      NotificationType = "deployment.stopped"
	NotifNodeOffline            NotificationType = "node.offline"
	NotifNodeOnline             NotificationType = "node.online"
	NotifBillingInvoice         NotificationType = "billing.invoice"
	NotifMarketplaceOrder       NotificationType = "marketplace.order"
)

type Notification struct {
	Type    NotificationType `json:"type"`
	UserID  string           `json:"user_id"`
	Title   string           `json:"title"`
	Message string           `json:"message"`
	Data    interface{}       `json:"data,omitempty"`
}
