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

ActiveRecord::Schema.define(version: 20180515060328) do

  create_table "accounts", force: :cascade do |t|
    t.binary  "address",      limit: 20, null: false
    t.integer "block_number", limit: 8,  null: false
    t.string  "balance",      limit: 32, null: false
    t.integer "nonce",        limit: 8,  null: false
  end

  add_index "accounts", ["address", "block_number"], name: "index_accounts_on_address_and_block_number", unique: true, using: :btree
  add_index "accounts", ["address"], name: "index_accounts_on_address", using: :btree

  create_table "block_headers", force: :cascade do |t|
    t.binary  "hash",         limit: 32,   null: false
    t.binary  "parent_hash",  limit: 32,   null: false
    t.binary  "uncle_hash",   limit: 32,   null: false
    t.binary  "coinbase",     limit: 20,   null: false
    t.binary  "root",         limit: 32,   null: false
    t.binary  "tx_hash",      limit: 32,   null: false
    t.binary  "receipt_hash", limit: 32,   null: false
    t.integer "difficulty",   limit: 8,    null: false
    t.integer "number",       limit: 8,    null: false
    t.integer "gas_limit",    limit: 8,    null: false
    t.integer "gas_used",     limit: 8,    null: false
    t.integer "time",         limit: 8,    null: false
    t.binary  "extra_data",   limit: 1024
    t.binary  "mix_digest",   limit: 32,   null: false
    t.binary  "nonce",        limit: 8,    null: false
  end

  add_index "block_headers", ["hash"], name: "index_block_headers_on_hash", unique: true, using: :btree
  add_index "block_headers", ["number"], name: "index_block_headers_on_number", unique: true, using: :btree

  create_table "erc20", force: :cascade do |t|
    t.binary  "address",      limit: 20,       null: false
    t.integer "block_number", limit: 8,        null: false
    t.binary  "code",         limit: 16777215, null: false
    t.string  "name",         limit: 32
    t.string  "total_supply", limit: 32
    t.integer "decimals",     limit: 8
  end

  add_index "erc20", ["address"], name: "index_erc20_on_address", unique: true, using: :btree
  add_index "erc20", ["block_number"], name: "index_erc20_on_block_number", using: :btree

  create_table "total_difficulty", force: :cascade do |t|
    t.integer "block", limit: 8,   null: false
    t.binary  "hash",  limit: 32,  null: false
    t.string  "td",    limit: 255, null: false
  end

  add_index "total_difficulty", ["hash"], name: "index_total_difficulty_on_hash", unique: true, using: :btree

  create_table "transaction_receipts", force: :cascade do |t|
    t.binary  "root",                limit: 32
    t.integer "status",              limit: 1
    t.integer "cumulative_gas_used", limit: 8,               null: false
    t.binary  "bloom",               limit: 256,             null: false
    t.binary  "tx_hash",             limit: 32,              null: false
    t.binary  "contract_address",    limit: 20
    t.integer "gas_used",            limit: 8,               null: false
    t.integer "block_number",        limit: 8,   default: 0, null: false
  end

  add_index "transaction_receipts", ["block_number"], name: "index_transaction_receipts_on_block_number", using: :btree
  add_index "transaction_receipts", ["tx_hash"], name: "index_transaction_receipts_on_tx_hash", unique: true, using: :btree

  create_table "transactions", force: :cascade do |t|
    t.binary  "hash",         limit: 32,                   null: false
    t.binary  "block_hash",   limit: 32,                   null: false
    t.binary  "from",         limit: 20,                   null: false
    t.binary  "to",           limit: 20
    t.integer "nonce",        limit: 8,                    null: false
    t.string  "gas_price",    limit: 32,                   null: false
    t.integer "gas_limit",    limit: 8,                    null: false
    t.string  "amount",       limit: 32,                   null: false
    t.binary  "payload",      limit: 16777215,             null: false
    t.integer "block_number", limit: 8,        default: 0, null: false
  end

  add_index "transactions", ["block_hash"], name: "index_transactions_on_block_hash", using: :btree
  add_index "transactions", ["block_number"], name: "index_transactions_on_block_number", using: :btree
  add_index "transactions", ["hash"], name: "index_transactions_on_hash", unique: true, using: :btree

end
