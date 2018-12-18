class AddGroupToAccountsTable < ActiveRecord::Migration[5.2]
  def change
    add_column :accounts, :group, :bigint, :default => 0
    add_index :accounts, [:group, :block_number]

    erc20_balance_table = select_all("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.tables WHERE TABLE_NAME LIKE 'erc20_balance_%'")
    erc20_balance_table.each do |row|
      add_column(row['TABLE_NAME'], :group, :bigint, :default => 0)
      add_index(row['TABLE_NAME'], [:group, :block_number])
    end
  end
end
