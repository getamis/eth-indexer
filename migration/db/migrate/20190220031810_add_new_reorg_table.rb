class AddNewReorgTable < ActiveRecord::Migration[5.2]
  def change
    # Create new table for reorg events
    create_table :reorgs2 do |t|
      t.integer :from, :limit => 8, :null => false
      t.binary :from_hash, :limit => 32, :null => false
      t.integer :to, :limit => 8, :null => false
      t.binary :to_hash, :limit => 32, :null => false
      t.datetime :created_at, :null => false
    end
    add_index :reorgs2, [:from, :to]
    add_index :reorgs2, [:from_hash, :to_hash], :unique => true
  end
end
