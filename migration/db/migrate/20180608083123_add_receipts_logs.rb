class AddReceiptsLogs < ActiveRecord::Migration[5.2]
  def change
    create_table :receipt_logs do |t|
      t.binary :tx_hash, :limit => 32, :null => false
      t.integer :block_number, :limit => 8, :null => false
      t.binary :contract_address, :limit => 20, :null => false
      t.binary :event_name, :limit => 32, :null => false
      t.binary :topic1, :limit => 32, :null => true
      t.binary :topic2, :limit => 32, :null => true
      t.binary :topic3, :limit => 32, :null => true
      t.binary :data, :limit => 1.megabyte, :null => false
    end
    add_index :receipt_logs, :tx_hash
    add_index :receipt_logs, :block_number
    add_index :receipt_logs, [:block_number, :contract_address]
    add_index :receipt_logs, [:block_number, :contract_address, :event_name], :name => 'index_receipt_logs_on_nae'
  end
end
