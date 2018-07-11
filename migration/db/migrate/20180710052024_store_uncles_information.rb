class StoreUnclesInformation < ActiveRecord::Migration[5.2]
  def change
    create_table :uncle_headers do |t|
      t.integer :position, :limit => 1, :null => false
      t.string :reward, :limit => 32, :null => false
      t.integer :block_number, :limit => 8, :null => false
      t.binary :uncle_hash, :limit => 32, :null => false
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
    add_index :uncle_headers, :hash, :unique => true
  end
end
