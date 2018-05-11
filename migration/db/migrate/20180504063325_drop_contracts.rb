class DropContracts < ActiveRecord::Migration
  def change
    drop_table :contracts

    rename_table :contract_code, :erc20
    change_column :erc20, :code, :binary, :limit => 1.megabyte, :null => false
    remove_column :erc20, :hash
    add_column :erc20, :name, :string, :limit => 8
    add_column :erc20, :total_supply, :string, :limit => 32
    add_column :erc20, :decimals, :integer, :limit => 8
  end
end
