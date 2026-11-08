package model

// SubscriptionAccount adalah pasangan user + akun broker yang berhak menerima
// sinyal satu simbol. Tipe ini bukan tabel; ia hanya bentuk hasil query yang
// dipakai Runner untuk fan-out per akun.
type SubscriptionAccount struct {
	User    User
	Account BrokerAccount
}
