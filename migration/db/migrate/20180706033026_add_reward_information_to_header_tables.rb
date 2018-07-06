class AddRewardInformationToHeaderTables < ActiveRecord::Migration[5.2]
  def change
    add_column :block_headers, :miner_reward, :string, :limit => 32, :null => false
    add_column :block_headers, :uncles_inclusion_reward, :string, :limit => 32, :null => false
    add_column :block_headers, :txs_fee, :string, :limit => 32, :null => false
    add_column :block_headers, :uncles_reward, :string, :limit => 32, :null => true
    add_column :block_headers, :uncle1_hash, :binary, :limit => 32, :null => true
    add_column :block_headers, :uncle2_hash, :binary, :limit => 32, :null => true
  end
end
