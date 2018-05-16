class ModifiedErc20NameSize < ActiveRecord::Migration
  def change
    change_column :erc20, :name, :string, :limit => 32
  end
end
