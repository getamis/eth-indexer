class AddReorgs < ActiveRecord::Migration[5.2]
  def change
    create_table :reorgs do |t|
      t.integer :from, :limit => 8, :null => false
      t.binary :from_hash, :limit => 32, :null => false
      t.integer :to, :limit => 8, :null => false
      t.binary :to_hash, :limit => 32, :null => false
      t.datetime :created_at, :null => false
    end
    add_index :reorgs, [:from, :to]
  end
end
