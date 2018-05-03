class AddBlockNumber < ActiveRecord::Migration
  def change
    add_column :transactions, :block_number, :integer, :limit => 8, null: false, :default => 0
    add_index :transactions, :block_number
    add_column :transaction_receipts, :block_number, :integer, :limit => 8, null: false, :default => 0
    add_index :transaction_receipts, :block_number
  end
end
