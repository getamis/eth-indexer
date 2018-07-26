class ChangeGpToBeBigInt < ActiveRecord::Migration[5.2]
  def up
    change_column :transactions, :gas_price, :integer, :limit => 8
  end

  def down
    change_column :transactions, :gas_price, :string, :limit => 32
  end
end
