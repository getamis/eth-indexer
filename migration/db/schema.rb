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

ActiveRecord::Schema.define(version: 2018_06_20_072545) do

  create_table "accounts", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "address", limit: 20, null: false
    t.bigint "block_number", null: false
    t.string "balance", limit: 32, null: false
    t.index ["address", "block_number"], name: "index_accounts_on_address_and_block_number", unique: true
    t.index ["address"], name: "index_accounts_on_address"
  end

  create_table "block_headers", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "hash", limit: 32, null: false
    t.binary "parent_hash", limit: 32, null: false
    t.binary "uncle_hash", limit: 32, null: false
    t.binary "coinbase", limit: 20, null: false
    t.binary "root", limit: 32, null: false
    t.binary "tx_hash", limit: 32, null: false
    t.binary "receipt_hash", limit: 32, null: false
    t.bigint "difficulty", null: false
    t.bigint "number", null: false
    t.bigint "gas_limit", null: false
    t.bigint "gas_used", null: false
    t.bigint "time", null: false
    t.binary "extra_data", limit: 1024
    t.binary "mix_digest", limit: 32, null: false
    t.binary "nonce", limit: 8, null: false
    t.index ["hash"], name: "index_block_headers_on_hash", unique: true
    t.index ["number"], name: "index_block_headers_on_number", unique: true
  end

  create_table "erc20", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "address", limit: 20, null: false
    t.bigint "block_number", null: false
    t.binary "code", limit: 16777215, null: false
    t.string "name", limit: 32
    t.string "total_supply", limit: 32
    t.bigint "decimals"
    t.index ["address"], name: "index_erc20_on_address", unique: true
    t.index ["block_number"], name: "index_erc20_on_block_number"
  end

  create_table "eth_transfer", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "tx_hash", limit: 32, null: false
    t.bigint "block_number", null: false
    t.binary "from", limit: 20, null: false
    t.binary "to", limit: 20, null: false
    t.string "value", limit: 32, null: false
    t.index ["block_number"], name: "index_eth_transfer_on_block_number"
    t.index ["from"], name: "index_eth_transfer_on_from"
    t.index ["to"], name: "index_eth_transfer_on_to"
    t.index ["tx_hash"], name: "index_eth_transfer_on_tx_hash"
  end

  create_table "receipt_logs", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "tx_hash", limit: 32, null: false
    t.bigint "block_number", null: false
    t.binary "contract_address", limit: 20, null: false
    t.binary "event_name", limit: 32, null: false
    t.binary "topic1", limit: 32
    t.binary "topic2", limit: 32
    t.binary "topic3", limit: 32
    t.binary "data", limit: 16777215, null: false
    t.index ["block_number", "contract_address", "event_name"], name: "index_receipt_logs_on_nae"
    t.index ["block_number", "contract_address"], name: "index_receipt_logs_on_block_number_and_contract_address"
    t.index ["block_number"], name: "index_receipt_logs_on_block_number"
    t.index ["tx_hash"], name: "index_receipt_logs_on_tx_hash"
  end

  create_table "subscriptions", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.bigint "block_number", default: 0
    t.bigint "group", null: false
    t.binary "address", limit: 20, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["address"], name: "index_subscriptions_on_address", unique: true
    t.index ["block_number"], name: "index_subscriptions_on_block_number"
    t.index ["group"], name: "index_subscriptions_on_group"
  end

  create_table "total_balances", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.bigint "block_number", null: false
    t.binary "token", limit: 20, null: false
    t.bigint "group", null: false
    t.string "balance", limit: 32, null: false
    t.index ["block_number", "token", "group"], name: "index_total_balances_on_block_number_and_token_and_group", unique: true
    t.index ["block_number"], name: "index_total_balances_on_block_number"
    t.index ["group"], name: "index_total_balances_on_group"
    t.index ["token", "group"], name: "index_total_balances_on_token_and_group"
    t.index ["token"], name: "index_total_balances_on_token"
  end

  create_table "total_difficulty", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.bigint "block", null: false
    t.binary "hash", limit: 32, null: false
    t.string "td", null: false
    t.index ["hash"], name: "index_total_difficulty_on_hash", unique: true
  end

  create_table "transaction_receipts", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "root", limit: 32
    t.integer "status", limit: 1
    t.bigint "cumulative_gas_used", null: false
    t.binary "bloom", limit: 256, null: false
    t.binary "tx_hash", limit: 32, null: false
    t.binary "contract_address", limit: 20
    t.bigint "gas_used", null: false
    t.bigint "block_number", null: false
    t.index ["block_number"], name: "index_transaction_receipts_on_block_number"
    t.index ["tx_hash"], name: "index_transaction_receipts_on_tx_hash", unique: true
  end

  create_table "transactions", options: "ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci", force: :cascade do |t|
    t.binary "hash", limit: 32, null: false
    t.binary "block_hash", limit: 32, null: false
    t.binary "from", limit: 20, null: false
    t.binary "to", limit: 20
    t.bigint "nonce", null: false
    t.string "gas_price", limit: 32, null: false
    t.bigint "gas_limit", null: false
    t.string "amount", limit: 32, null: false
    t.binary "payload", limit: 16777215, null: false
    t.bigint "block_number", null: false
    t.index ["block_hash"], name: "index_transactions_on_block_hash"
    t.index ["block_number"], name: "index_transactions_on_block_number"
    t.index ["hash"], name: "index_transactions_on_hash", unique: true
  end

end
