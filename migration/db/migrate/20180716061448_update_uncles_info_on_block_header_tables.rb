class UpdateUnclesInfoOnBlockHeaderTables < ActiveRecord::Migration[5.2]
  def change
      remove_column :block_headers, :uncles_reward, :string, :limit => 32, :null => true
      add_column :block_headers, :uncle1_reward, :string, :limit => 32, :null => true
      add_column :block_headers, :uncle1_coinbase, :binary, :limit => 20, :null => true
      add_column :block_headers, :uncle2_reward, :string, :limit => 32, :null => true
      add_column :block_headers, :uncle2_coinbase, :binary, :limit => 20, :null => true
    end
end
