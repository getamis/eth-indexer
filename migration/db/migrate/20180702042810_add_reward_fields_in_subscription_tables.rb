class AddRewardFieldsInSubscriptionTables < ActiveRecord::Migration[5.2]
  def change
    add_column :total_balances, :uncles_reward, :string, :limit => 32, :null => false
    add_column :total_balances, :miner_reward, :string, :limit => 32, :null => false
  end
end
