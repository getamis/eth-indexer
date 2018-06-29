class AddSubscriptionTable < ActiveRecord::Migration[5.2]
  def change
    create_table :subscriptions do |t|
      t.integer :block_number, :limit => 8, :default => 0
      t.integer :group, :limit => 8, :null => false
      t.binary :address, :limit => 20, :null => false
      t.timestamps :null => false
    end
    add_index :subscriptions, :block_number
    add_index :subscriptions, :address, :unique => true
    add_index :subscriptions, :group

    create_table :total_balances do |t|
      t.integer :block_number, :limit => 8, :null => false
      t.binary :token, :limit => 20, :null => false
      t.integer :group, :limit => 8, :null => false
      t.string :balance, :limit => 32, :null => false
      t.string :tx_fee, :limit => 32, :null => false
    end
    add_index :total_balances, :block_number
    add_index :total_balances, :token
    add_index :total_balances, :group
    add_index :total_balances, [:token, :group]
    add_index :total_balances, [:block_number, :token, :group], :unique => true
  end
end
