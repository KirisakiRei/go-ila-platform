// Module ila-strategy memuat aturan trading ILA 2.0 sebagai pustaka murni.
//
// Pustaka ini TIDAK tahu soal database, gRPC, broker, atau HTTP. Isinya hanya
// keputusan trading: di mana sweep, di mana CISD, di mana entry, SL, dan TP.
// Siapa pun yang butuh aturan itu cukup memberi deret candle.
//
// Module ini adalah SALINAN dari go-trade/pkg. Engine live di VPS 1 tetap
// memakai salinannya sendiri; keduanya harus identik. Lihat ASAL.md dan
// scripts/cek-ila-strategy.sh.
module ila-strategy

go 1.24.1
