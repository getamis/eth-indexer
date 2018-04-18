class InitializeIndexerTables < ActiveRecord::Migration
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
    end
    add_index :transactions, :hash, :unique => true
    add_index :transactions, :block_hash

    create_table :transaction_receipts do |t|
      t.binary :root, :limit => 32
      t.integer :status, :limit => 1
      t.integer :cumulative_gas_used, :limit => 8, :null => false
      t.binary :bloom, :limit => 256, :null => false
      t.binary :tx_hash, :limit => 32, :null => false
      t.binary :contract_address, :limit => 20
      t.integer :gas_used, :limit => 8, :null => false
    end
    add_index :transaction_receipts, :tx_hash, :unique => true

    # Use state_blocks to record the blocks for which state (accounts and contracts table) are updated.
    create_table :state_blocks do |t|
      t.integer :number, :limit => 8, :null => false
    end
    add_index :state_blocks, :number, :unique => true

    create_table :contract_code do |t|
      t.binary :address, :limit => 20, :null => false
      t.binary :hash, :limit => 32, :null => false
      t.text :code, :limit => 1.megabyte, :null => false
    end
    add_index :contract_code, :address, :unique => true

    create_table :accounts do |t|
      t.binary :address, :limit => 20, :null => false
      t.integer :block_number, :limit => 8, :null => false
      t.string :balance, :limit => 32, :null => false
      t.integer :nonce, :limit => 8, :null => false
    end
    add_index :accounts, :address
    add_index :accounts, [:address, :block_number], :unique => true

    create_table :contracts do |t|
      t.binary :address, :limit => 20, :null => false
      t.integer :block_number, :limit => 8, :null => false
      t.string :balance, :limit => 32, :null => false
      t.integer :nonce, :limit => 8, :null => false
      t.binary :root, :limit => 32, :null => false
      t.binary :storage, :limit => 10.megabyte, :null => false
    end
    add_index :contracts, :address
    add_index :contracts, [:address, :block_number], :unique => true

    # TODO: Add foreign keys?
  end
end
