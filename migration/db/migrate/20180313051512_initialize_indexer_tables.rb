class InitializeIndexerTables < ActiveRecord::Migration[5.2]
  def change
    create_table :block_headers do |t|
      t.binary :hash, :limit => 32, :null => false
      t.binary :parent_hash, :limit => 32, :null => false
      t.binary :uncle_hash, :limit => 32, :null => false
      t.binary :coinbase, :limit => 20, :null => false
      t.binary :root, :limit => 32, :null => false
      t.binary :tx_hash, :limit => 32, :null => false
      t.binary :receipt_hash, :limit => 32, :null => false
      t.integer :difficulty, :limit => 8, :null => false
      t.integer :number, :limit => 8, :null => false
      t.integer :gas_limit, :limit => 8, :null => false
      t.integer :gas_used, :limit => 8, :null => false
      t.integer :time, :limit => 8, :null => false
      t.binary :extra_data, :limit => 1024
      t.binary :mix_digest, :limit => 32, :null => false
      t.binary :nonce, :limit => 8, :null => false
    end
    add_index :block_headers, :hash, :unique => true
    add_index :block_headers, :number, :unique => true

    create_table :transactions do |t|
      t.binary :hash, :limit => 32, :null => false
      t.binary :block_hash, :limit => 32, :null => false
      t.binary :from, :limit => 20, :null => false
      t.binary :to, :limit => 20
      t.integer :nonce, :limit => 8, :null => false
      t.string :gas_price, :limit => 32, :null => false
      t.integer :gas_limit, :limit => 8, :null => false
      t.string :amount, :limit => 32, :null => false
      t.binary :payload, :limit => 1.megabyte, :null => false
      t.integer :block_number, :limit => 8, :null => false
    end
    add_index :transactions, :hash, :unique => true
    add_index :transactions, :block_hash
    add_index :transactions, :block_number

    create_table :transaction_receipts do |t|
      t.binary :root, :limit => 32
      t.integer :status, :limit => 1
      t.integer :cumulative_gas_used, :limit => 8, :null => false
      t.binary :bloom, :limit => 256, :null => false
      t.binary :tx_hash, :limit => 32, :null => false
      t.binary :contract_address, :limit => 20
      t.integer :gas_used, :limit => 8, :null => false
      t.integer :block_number, :limit => 8, :null => false
    end
    add_index :transaction_receipts, :tx_hash, :unique => true
    add_index :transaction_receipts, :block_number

    create_table :total_difficulty do |t|
      t.integer :block, :limit => 8, :null => false
      t.binary :hash, :limit => 32, :null => false
      t.string :td, null: false
    end
    add_index :total_difficulty, :hash, :unique => true

    create_table :erc20 do |t|
      t.binary :address, :limit => 20, :null => false
      t.integer :block_number, :limit => 8, :null => false
      t.binary :code, :limit => 1.megabyte, :null => false
      t.string :name, :limit => 32
      t.string :total_supply, :limit => 32
      t.integer :decimals, :limit => 8
    end
    add_index :erc20, :address, :unique => true
    add_index :erc20, :block_number

    create_table :accounts do |t|
      t.binary :address, :limit => 20, :null => false
      t.integer :block_number, :limit => 8, :null => false
      t.string :balance, :limit => 32, :null => false
      t.integer :nonce, :limit => 8, :null => false
    end
    add_index :accounts, :address
    add_index :accounts, [:address, :block_number], :unique => true

  end
end
