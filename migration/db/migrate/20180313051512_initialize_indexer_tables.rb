class InitializeIndexerTables < ActiveRecord::Migration
  def change
    create_table :block_headers do |t|
      t.string :parent_hash, :null => false
      t.string :uncle_hash, :null => false
      t.string :coinbase, :null => false
      t.string :root, :null => false
      t.string :tx_hash, :null => false
      t.string :receipt_hash, :null => false
      t.binary :bloom
      t.integer :difficulty, :limit => 8, :null => false
      t.integer :number, :limit => 8, :null => false
      t.integer :gas_limit, :limit => 8, :null => false
      t.integer :gas_used, :limit => 8, :null => false
      t.binary :extra_data
      t.binary :nonce, :limit => 8, :null => false
    end
    # TODO: Add indexes to block_headers
    
    create_table :transactions do |t|
      t.integer :nonce, :limit => 8, :null => false
      t.integer :price, :limit => 8, :null => false
      t.integer :gas_limit, :limit => 8, :null => false
      t.string :recipient
      t.integer :amount, :limit => 8, :null => false
      t.binary :payload
      t.integer :v, :limit => 8, :null => false
      t.integer :r, :limit => 8, :null => false
      t.integer :s, :limit => 8, :null => false
      t.string :hash, :null => false
    end
    # TODO: Add indexes to transactions
  
    create_table :transaction_recepits do |t|
      t.binary :root, :null => false
      t.integer :status, :null => false
      t.integer :cumulative_gas_used, :limit => 8, :null => false
      t.binary :bloom
      t.string :tx_hash, :null => false
      t.string :contract_address
      t.integer :gas_used, :limit => 8, :null => false
    end
    # TODO: Add indexes to transaction_receipts

    create_table :transaction_logs do |t|
      t.string :tx_hash, :null => false
      t.integer :tx_index, :null => false
      t.string :block_hash, :null => false
      t.integer :log_index, :null => false
      t.boolean :removed
      t.string
      t.string :address
      t.string :topic_0
      t.string :topic_1
      t.string :topic_2
      t.binary :data
      t.integer :block_number, :limit => 8, :null => false
    end
    # TODO: Add indexes to transaction_logs
  end
end
