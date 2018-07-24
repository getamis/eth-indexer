class RemoveErc20Code < ActiveRecord::Migration[5.2]
  def change
    remove_column :erc20, :code, :binary
  end
end
