class RemoveTdFromHeader < ActiveRecord::Migration
  def change
    remove_column :block_headers, :td
  end
end
