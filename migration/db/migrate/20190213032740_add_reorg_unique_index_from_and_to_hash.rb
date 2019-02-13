class AddReorgUniqueIndexFromAndToHash < ActiveRecord::Migration[5.2]
  def change
    add_index :reorgs, [:from_hash, :to_hash], :unique => true
  end
end
