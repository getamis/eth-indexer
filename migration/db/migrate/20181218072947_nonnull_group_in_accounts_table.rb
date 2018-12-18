class NonnullGroupInAccountsTable < ActiveRecord::Migration[5.2]
  def change
    change_column_null :accounts, :group, false
    change_column_default :accounts, :group, from: 0, to: nil

    erc20_balance_table = select_all("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.tables WHERE TABLE_NAME LIKE 'erc20_balance_%'")
    erc20_balance_table.each do |row|
      change_column_null(row['TABLE_NAME'], :group, false)
      change_column_default(row['TABLE_NAME'], :group, from: 0, to: nil)
    end
  end
end
