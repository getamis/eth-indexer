class AddCreatedAtInHeader < ActiveRecord::Migration[5.2]
  def change
    add_column :block_headers, :created_at, :datetime, :null => false
    add_index :block_headers, :created_at
  end
end
