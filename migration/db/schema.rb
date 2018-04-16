# encoding: UTF-8
# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# Note that this schema.rb definition is the authoritative source for your
# database schema. If you need to create the application database on another
# system, you should be using db:schema:load, not running all the migrations
# from scratch. The latter is a flawed and unsustainable approach (the more migrations
# you'll amass, the slower it'll run and the greater likelihood for issues).
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema.define(version: 20180313051512) do

  create_table "block_headers", force: :cascade do |t|
    t.string  "hash",         limit: 64,    null: false
    t.string  "parent_hash",  limit: 64,    null: false
    t.string  "uncle_hash",   limit: 64,    null: false
    t.string  "coinbase",     limit: 40,    null: false
    t.string  "root",         limit: 64,    null: false
    t.string  "tx_hash",      limit: 64,    null: false
    t.string  "receipt_hash", limit: 64,    null: false
    t.binary  "bloom",        limit: 65535
    t.integer "difficulty",   limit: 8,     null: false
    t.integer "number",       limit: 8,     null: false
    t.integer "gas_limit",    limit: 8
    t.integer "gas_used",     limit: 8
    t.integer "time",         limit: 8
    t.binary  "extra_data",   limit: 65535
    t.string  "mix_digest",   limit: 255
    t.binary  "nonce",        limit: 65535
  end

  add_index "block_headers", ["number"], name: "index_block_headers_on_number", unique: true, using: :btree

  create_table "transaction_receipts", force: :cascade do |t|
    t.binary  "root",                limit: 64
    t.integer "status",              limit: 4
    t.integer "cumulative_gas_used", limit: 8
    t.binary  "bloom",               limit: 65535
    t.string  "tx_hash",             limit: 64
    t.string  "contract_address",    limit: 40
    t.integer "gas_used",            limit: 8
  end

  add_index "transaction_receipts", ["tx_hash"], name: "index_transaction_receipts_on_tx_hash", unique: true, using: :btree

  create_table "transactions", force: :cascade do |t|
    t.string  "hash",       limit: 64
    t.string  "block_hash", limit: 64
    t.string  "from",       limit: 40
    t.string  "to",         limit: 40
    t.binary  "nonce",      limit: 65535
    t.integer "gas_price",  limit: 8
    t.integer "gas_limit",  limit: 8
    t.integer "amount",     limit: 8
    t.binary  "payload",    limit: 65535
    t.integer "v",          limit: 8
    t.integer "s",          limit: 8
    t.integer "r",          limit: 8
  end

  add_index "transactions", ["hash"], name: "index_transactions_on_hash", unique: true, using: :btree

end
