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

  create_table "account_balances", force: :cascade do |t|
    t.string  "address",          limit: 255, null: false
    t.integer "token_id",         limit: 4,   null: false
    t.integer "balance",          limit: 8,   null: false
    t.integer "evaluated_height", limit: 8,   null: false
  end

  create_table "accounts", force: :cascade do |t|
    t.string "address", limit: 255,   null: false
    t.binary "code",    limit: 65535
  end

  create_table "block_headers", force: :cascade do |t|
    t.string  "parent_hash",  limit: 255,   null: false
    t.string  "uncle_hash",   limit: 255,   null: false
    t.string  "coinbase",     limit: 255,   null: false
    t.string  "root",         limit: 255,   null: false
    t.string  "tx_hash",      limit: 255,   null: false
    t.string  "receipt_hash", limit: 255,   null: false
    t.binary  "bloom",        limit: 65535
    t.integer "difficulty",   limit: 8,     null: false
    t.integer "number",       limit: 8,     null: false
    t.integer "gas_limit",    limit: 8,     null: false
    t.integer "gas_used",     limit: 8,     null: false
    t.binary  "extra_data",   limit: 65535
    t.binary  "nonce",        limit: 8,     null: false
  end

  create_table "tokens", force: :cascade do |t|
    t.string "contract_address", limit: 255
    t.string "type",             limit: 255, null: false
    t.string "full_name",        limit: 255, null: false
  end

  create_table "transaction_logs", force: :cascade do |t|
    t.string  "tx_hash",      limit: 255,   null: false
    t.integer "tx_index",     limit: 4,     null: false
    t.string  "block_hash",   limit: 255,   null: false
    t.integer "log_index",    limit: 4,     null: false
    t.boolean "removed"
    t.string  "address",      limit: 255
    t.string  "topic_0",      limit: 255
    t.string  "topic_1",      limit: 255
    t.string  "topic_2",      limit: 255
    t.binary  "data",         limit: 65535
    t.integer "block_number", limit: 8,     null: false
  end

  create_table "transaction_recepits", force: :cascade do |t|
    t.binary  "root",                limit: 65535, null: false
    t.integer "status",              limit: 4,     null: false
    t.integer "cumulative_gas_used", limit: 8,     null: false
    t.binary  "bloom",               limit: 65535
    t.string  "tx_hash",             limit: 255,   null: false
    t.string  "contract_address",    limit: 255
    t.integer "gas_used",            limit: 8,     null: false
  end

  create_table "transactions", force: :cascade do |t|
    t.integer "nonce",     limit: 8,     null: false
    t.integer "price",     limit: 8,     null: false
    t.integer "gas_limit", limit: 8,     null: false
    t.string  "recipient", limit: 255
    t.integer "amount",    limit: 8,     null: false
    t.binary  "payload",   limit: 65535
    t.integer "v",         limit: 8,     null: false
    t.integer "r",         limit: 8,     null: false
    t.integer "s",         limit: 8,     null: false
    t.string  "hash",      limit: 255,   null: false
  end

end
