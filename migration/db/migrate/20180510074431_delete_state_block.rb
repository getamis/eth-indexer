class DeleteStateBlock < ActiveRecord::Migration
  def change
    drop_table :state_blocks
  end
end
